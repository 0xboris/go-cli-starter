package browser

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// fakeCommand re-runs this test binary as TestHelperProcess instead of the real
// program, so tests never launch a browser (the pattern gh's git package uses).
func fakeCommand(t *testing.T, exitCode int, stdout, stderr string, gotArgs *[]string) commandCtx {
	t.Helper()
	return func(ctx context.Context, name string, args ...string) *exec.Cmd {
		*gotArgs = append([]string{name}, args...)
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestHelperProcess", "--")
		cmd.Env = append(os.Environ(),
			"GO_WANT_HELPER_PROCESS=1",
			fmt.Sprintf("HELPER_EXIT=%d", exitCode),
			"HELPER_STDOUT="+stdout,
			"HELPER_STDERR="+stderr,
		)
		return cmd
	}
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	fmt.Fprint(os.Stdout, os.Getenv("HELPER_STDOUT"))
	fmt.Fprint(os.Stderr, os.Getenv("HELPER_STDERR"))
	code := 0
	fmt.Sscanf(os.Getenv("HELPER_EXIT"), "%d", &code)
	os.Exit(code)
}

// self is a program that exists on every OS (this test binary) so LookPath
// succeeds; the fake command factory replaces what actually runs.
func self(t *testing.T) string {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Skip("cannot locate test binary:", err)
	}
	return exe
}

func TestBrowsePassesURLLast(t *testing.T) {
	var got []string
	var out bytes.Buffer
	l := &Launcher{Command: []string{self(t), "--flag"}, Stdout: &out, Stderr: &out,
		commandContext: fakeCommand(t, 0, "opened\n", "", &got)}

	if err := l.Browse(context.Background(), "https://example.com/items/3"); err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0] != self(t) || got[1] != "--flag" || got[2] != "https://example.com/items/3" {
		t.Fatalf("args = %q", got)
	}
	if out.String() != "opened\n" {
		t.Fatalf("child stdout went to %q, want the injected writer", out.String())
	}
}

func TestBrowseFailureCarriesExitCodeAndStderr(t *testing.T) {
	var got []string
	l := &Launcher{Command: []string{self(t)}, commandContext: fakeCommand(t, 3, "", "no display\n", &got)}

	err := l.Browse(context.Background(), "https://example.com")
	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode != 3 || !strings.Contains(exitErr.Stderr, "no display") {
		t.Fatalf("error = %v, want ExitError{ExitCode: 3, Stderr: no display}", err)
	}
}

func TestBrowseMissingProgram(t *testing.T) {
	l := New([]string{"definitely-not-a-real-launcher"}, nil, nil)
	var nie *NotInstalledError
	if err := l.Browse(context.Background(), "https://example.com"); !errors.As(err, &nie) {
		t.Fatalf("error = %v, want NotInstalledError", err)
	}
}

func TestBrowseRejectsOptionLikeURL(t *testing.T) {
	var got []string
	l := &Launcher{Command: []string{self(t)}, commandContext: fakeCommand(t, 0, "", "", &got)}
	if err := l.Browse(context.Background(), "--help"); err == nil || got != nil {
		t.Fatalf("error = %v, ran %q; want refusal without running", err, got)
	}
}

func TestDefaultCommand(t *testing.T) {
	for goos, want := range map[string]string{"darwin": "open", "linux": "xdg-open", "windows": "rundll32"} {
		if got := DefaultCommand(goos); got[0] != want {
			t.Errorf("DefaultCommand(%q) = %q, want %q first", goos, got, want)
		}
	}
}
