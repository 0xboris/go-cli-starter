// Package iostreams owns stdin/stdout/stderr and every terminal capability question.
// Commands never touch os.Stdout directly; they write to IOStreams.Out (data) and
// IOStreams.ErrOut (everything else).
package iostreams

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/term"
)

const DefaultWidth = 80

// ErrClosedPagerPipe is returned when the user quits the pager early; the CLI exits 0.
type ErrClosedPagerPipe struct{ error }

type IOStreams struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer

	originalOut io.Writer

	stdinTTYOverride  bool
	stdinIsTTY        bool
	stdoutTTYOverride bool
	stdoutIsTTY       bool
	stderrTTYOverride bool
	stderrIsTTY       bool

	colorEnabled    bool
	neverPrompt     bool
	spinnerDisabled bool
	widthOverride   int

	pagerCommand string
	pagerProcess *os.Process

	progressMu    sync.Mutex
	progressStop  chan struct{}
	progressDone  chan struct{}
	progressLabel string
}

// System returns IOStreams wired to the real process streams, honoring
// NO_COLOR, CLICOLOR, CLICOLOR_FORCE and TOOL_FORCE_TTY.
func System() *IOStreams {
	s := &IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr, originalOut: os.Stdout}

	if spec := os.Getenv("TOOL_FORCE_TTY"); spec != "" {
		s.SetStdoutTTY(true)
		if w, err := strconv.Atoi(spec); err == nil { // numeric value = width in columns
			s.widthOverride = w
		}
	}
	forced := os.Getenv("CLICOLOR_FORCE") != "" && os.Getenv("CLICOLOR_FORCE") != "0"
	disabled := os.Getenv("NO_COLOR") != "" || os.Getenv("CLICOLOR") == "0" || os.Getenv("TERM") == "dumb"
	s.colorEnabled = forced || (!disabled && s.IsStdoutTTY())

	// Spinners only when both streams are terminals: never pollute logs or pipes.
	s.spinnerDisabled = !s.IsStdoutTTY() || !s.IsStderrTTY() || os.Getenv("TERM") == "dumb"
	return s
}

// Test returns IOStreams backed by buffers. All streams are non-TTY and colorless
// until the test opts in with SetStdoutTTY etc.
func Test() (*IOStreams, *bytes.Buffer, *bytes.Buffer, *bytes.Buffer) {
	in, out, errOut := &bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{}
	s := &IOStreams{In: in, Out: out, ErrOut: errOut, originalOut: out}
	s.SetStdinTTY(false)
	s.SetStdoutTTY(false)
	s.SetStderrTTY(false)
	s.spinnerDisabled = true
	return s, in, out, errOut
}

func isTerminal(v any) bool {
	f, ok := v.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

func (s *IOStreams) IsStdinTTY() bool {
	if s.stdinTTYOverride {
		return s.stdinIsTTY
	}
	return isTerminal(s.In)
}

func (s *IOStreams) IsStdoutTTY() bool {
	if s.stdoutTTYOverride {
		return s.stdoutIsTTY
	}
	return isTerminal(s.originalOut)
}

func (s *IOStreams) IsStderrTTY() bool {
	if s.stderrTTYOverride {
		return s.stderrIsTTY
	}
	return isTerminal(s.ErrOut)
}

func (s *IOStreams) SetStdinTTY(v bool)  { s.stdinTTYOverride, s.stdinIsTTY = true, v }
func (s *IOStreams) SetStdoutTTY(v bool) { s.stdoutTTYOverride, s.stdoutIsTTY = true, v }
func (s *IOStreams) SetStderrTTY(v bool) { s.stderrTTYOverride, s.stderrIsTTY = true, v }

func (s *IOStreams) ColorEnabled() bool        { return s.colorEnabled }
func (s *IOStreams) SetColorEnabled(v bool)    { s.colorEnabled = v }
func (s *IOStreams) SetNeverPrompt(v bool)     { s.neverPrompt = v }
func (s *IOStreams) SetSpinnerDisabled(v bool) { s.spinnerDisabled = v }
func (s *IOStreams) SetPager(cmd string)       { s.pagerCommand = cmd }

// CanPrompt is true only when stdin and stderr (the channel prompts are drawn on)
// are terminals and prompting has not been disabled (--no-input,
// TOOL_PROMPT_DISABLED, config). stdout may be redirected: `tool x > out.txt` can
// still ask, and the file only receives data.
func (s *IOStreams) CanPrompt() bool {
	if s.neverPrompt {
		return false
	}
	return s.IsStdinTTY() && s.IsStderrTTY()
}

// TerminalWidth returns the terminal width, or DefaultWidth when unknown.
func (s *IOStreams) TerminalWidth() int {
	if s.widthOverride > 0 {
		return s.widthOverride
	}
	if f, ok := s.originalOut.(*os.File); ok {
		if w, _, err := term.GetSize(int(f.Fd())); err == nil && w > 0 {
			return w
		}
	}
	return DefaultWidth
}

func (s *IOStreams) ColorScheme() *ColorScheme {
	return &ColorScheme{Enabled: s.colorEnabled}
}

// RunWithProgress shows a spinner on stderr while fn runs (TTY only).
func (s *IOStreams) RunWithProgress(label string, fn func() error) error {
	s.StartProgressIndicatorWithLabel(label)
	defer s.StopProgressIndicator()
	return fn()
}

func (s *IOStreams) StartProgressIndicatorWithLabel(label string) {
	if s.spinnerDisabled {
		return
	}
	s.progressMu.Lock()
	defer s.progressMu.Unlock()
	if s.progressStop != nil { // already running: just update the label
		s.progressLabel = label
		return
	}
	s.progressLabel = label
	s.progressStop, s.progressDone = make(chan struct{}), make(chan struct{})
	go func(stop, done chan struct{}) {
		defer close(done)
		frames := []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}
		t := time.NewTicker(120 * time.Millisecond)
		defer t.Stop()
		for i := 0; ; i++ {
			s.progressMu.Lock()
			_, _ = fmt.Fprintf(s.ErrOut, "\r\033[K%s %s", frames[i%len(frames)], s.progressLabel)
			s.progressMu.Unlock()
			select {
			case <-stop:
				_, _ = fmt.Fprint(s.ErrOut, "\r\033[K")
				return
			case <-t.C:
			}
		}
	}(s.progressStop, s.progressDone)
}

func (s *IOStreams) StopProgressIndicator() {
	s.progressMu.Lock()
	stop, done := s.progressStop, s.progressDone
	s.progressStop, s.progressDone = nil, nil
	s.progressMu.Unlock()
	if stop != nil {
		close(stop)
		<-done
	}
}

// StartPager pipes Out through the configured pager when stdout is a TTY.
// Failure to start a pager is a warning for the caller, never fatal.
func (s *IOStreams) StartPager() error {
	if s.pagerCommand == "" || s.pagerCommand == "cat" || !s.IsStdoutTTY() {
		return nil
	}
	args := strings.Fields(s.pagerCommand)
	cmd := exec.Command(args[0], args[1:]...)
	env := []string{}
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "PAGER=") {
			env = append(env, kv)
		}
	}
	if os.Getenv("LESS") == "" {
		env = append(env, "LESS=FRX")
	}
	cmd.Env = env
	cmd.Stdout = s.originalOut
	cmd.Stderr = s.ErrOut
	w, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	s.Out = &pagerWriter{w}
	s.pagerProcess = cmd.Process
	return nil
}

func (s *IOStreams) StopPager() {
	if s.pagerProcess == nil {
		return
	}
	if c, ok := s.Out.(io.Closer); ok {
		_ = c.Close()
	}
	_, _ = s.pagerProcess.Wait()
	s.pagerProcess = nil
	s.Out = s.originalOut
}

type pagerWriter struct{ io.WriteCloser }

func (w *pagerWriter) Write(p []byte) (int, error) {
	n, err := w.WriteCloser.Write(p)
	if err != nil && (errors.Is(err, io.ErrClosedPipe) || errors.Is(err, syscall.EPIPE)) {
		return n, &ErrClosedPagerPipe{err}
	}
	return n, err
}
