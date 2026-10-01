// Package prompter defines interactive prompts behind an interface so commands can
// be tested with a mock and implementations can be swapped (e.g. an accessible,
// line-based prompter, or charmbracelet/huh).
package prompter

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

// ErrInterrupt is returned when the user aborts a prompt (Ctrl-D / Ctrl-C).
// cmdutil.IsUserCancellation recognizes it, so the CLI exits 2 quietly.
var ErrInterrupt = errors.New("interrupted")

type Prompter interface {
	Input(prompt, defaultValue string) (string, error)
	Confirm(prompt string, defaultValue bool) (bool, error)
	// ConfirmDeletion makes the user type requiredValue to proceed.
	ConfirmDeletion(requiredValue string) error
}

// New returns a simple line-based prompter. It is accessible by construction
// (no screen redraws) and is a good default; swap in a richer one if needed.
func New(in io.Reader, out io.Writer) Prompter {
	return &linePrompter{in: bufio.NewReader(in), out: out}
}

type linePrompter struct {
	in  *bufio.Reader
	out io.Writer
}

func (p *linePrompter) readLine() (string, error) {
	line, err := p.in.ReadString('\n')
	if err != nil && !(errors.Is(err, io.EOF) && line != "") {
		if errors.Is(err, io.EOF) {
			return "", ErrInterrupt // Ctrl-D
		}
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func (p *linePrompter) Input(prompt, defaultValue string) (string, error) {
	if defaultValue != "" {
		fmt.Fprintf(p.out, "? %s (default: %s) ", prompt, defaultValue)
	} else {
		fmt.Fprintf(p.out, "? %s ", prompt)
	}
	v, err := p.readLine()
	if err != nil {
		return "", err
	}
	if v == "" {
		return defaultValue, nil
	}
	return v, nil
}

func (p *linePrompter) Confirm(prompt string, defaultValue bool) (bool, error) {
	hint := "y/N"
	if defaultValue {
		hint = "Y/n" // default in caps
	}
	fmt.Fprintf(p.out, "? %s (%s) ", prompt, hint)
	v, err := p.readLine()
	if err != nil {
		return false, err
	}
	switch strings.ToLower(v) {
	case "":
		return defaultValue, nil
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

func (p *linePrompter) ConfirmDeletion(requiredValue string) error {
	fmt.Fprintf(p.out, "? Type %s to confirm deletion: ", requiredValue)
	v, err := p.readLine()
	if err != nil {
		return err
	}
	if v != requiredValue {
		return fmt.Errorf("confirmation did not match: you entered %q", v)
	}
	return nil
}

// Mock is a hand-written mock (use `moq` to generate one for larger interfaces).
// Unset funcs fail loudly so tests notice unexpected prompts.
type Mock struct {
	InputFunc           func(prompt, defaultValue string) (string, error)
	ConfirmFunc         func(prompt string, defaultValue bool) (bool, error)
	ConfirmDeletionFunc func(requiredValue string) error
}

func (m *Mock) Input(p, d string) (string, error) {
	if m.InputFunc == nil {
		panic("unexpected Input prompt: " + p)
	}
	return m.InputFunc(p, d)
}

func (m *Mock) Confirm(p string, d bool) (bool, error) {
	if m.ConfirmFunc == nil {
		panic("unexpected Confirm prompt: " + p)
	}
	return m.ConfirmFunc(p, d)
}

func (m *Mock) ConfirmDeletion(v string) error {
	if m.ConfirmDeletionFunc == nil {
		panic("unexpected ConfirmDeletion prompt for: " + v)
	}
	return m.ConfirmDeletionFunc(v)
}
