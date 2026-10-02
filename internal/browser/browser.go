// Package browser opens URLs with an external program.
//
// Launcher is the template for wrapping any external program (a compiler, a
// renderer, git, ssh), modeled on gh's git.Client:
//   - one type per program, one method per operation, ctx first;
//   - the binary is resolved when first used, and a missing one is a typed error;
//   - streams are injected (never os.Std*); the child's stdout goes wherever the
//     composition root says (stderr here, because a launcher's chatter isn't data);
//   - a failure carries the exit code and stderr;
//   - the test seam is an unexported field, not a package-level variable.
package browser

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// Browser is what commands depend on.
type Browser interface {
	Browse(ctx context.Context, url string) error
}

// NotInstalledError means the launcher program could not be found.
type NotInstalledError struct {
	Name string
	Err  error
}

func (e *NotInstalledError) Error() string {
	return fmt.Sprintf("%s not found; set TOOL_BROWSER to a program that opens URLs", e.Name)
}
func (e *NotInstalledError) Unwrap() error { return e.Err }

// ExitError means the launcher ran and failed.
type ExitError struct {
	Name     string
	ExitCode int
	Stderr   string
	Err      error
}

func (e *ExitError) Error() string {
	msg := fmt.Sprintf("%s exited with status %d", e.Name, e.ExitCode)
	if s := strings.TrimSpace(e.Stderr); s != "" {
		msg += ": " + s
	}
	return msg
}
func (e *ExitError) Unwrap() error { return e.Err }

type commandCtx = func(ctx context.Context, name string, args ...string) *exec.Cmd

// Launcher runs Command followed by the URL.
type Launcher struct {
	Command []string // e.g. ["xdg-open"] or ["open"]; see DefaultCommand
	Stdout  io.Writer
	Stderr  io.Writer

	commandContext commandCtx // nil means exec.CommandContext
}

// New returns a Launcher. The composition root chooses the command (from env or
// config) and the streams; this package reads neither.
func New(command []string, stdout, stderr io.Writer) *Launcher {
	return &Launcher{Command: command, Stdout: stdout, Stderr: stderr}
}

// DefaultCommand is the platform's URL opener for a GOOS value.
func DefaultCommand(goos string) []string {
	switch goos {
	case "darwin":
		return []string{"open"}
	case "windows":
		return []string{"rundll32", "url.dll,FileProtocolHandler"}
	default:
		return []string{"xdg-open"}
	}
}

func (l *Launcher) Browse(ctx context.Context, url string) error {
	if len(l.Command) == 0 {
		return errors.New("no browser command configured")
	}
	if strings.HasPrefix(url, "-") {
		return fmt.Errorf("refusing to open %q: looks like an option", url)
	}
	name := l.Command[0]
	path, err := exec.LookPath(name) // never resolves from the current directory (exec.ErrDot)
	if err != nil {
		return &NotInstalledError{Name: name, Err: err}
	}

	commandContext := l.commandContext
	if commandContext == nil {
		commandContext = exec.CommandContext
	}
	args := append(append([]string{}, l.Command[1:]...), url)
	cmd := commandContext(ctx, path, args...)
	var stderr bytes.Buffer
	cmd.Stdout = l.Stdout
	cmd.Stderr = &stderr
	if l.Stderr != nil {
		cmd.Stderr = io.MultiWriter(&stderr, l.Stderr)
	}
	if err := cmd.Run(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return &ExitError{Name: name, ExitCode: exitErr.ExitCode(), Stderr: stderr.String(), Err: err}
		}
		return err
	}
	return nil
}
