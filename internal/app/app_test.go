package app

import (
	"os"
	"strings"
	"syscall"
	"testing"

	"example.com/tool/pkg/iostreams"
)

// End-to-end through Run: exit codes and stream routing.
func TestRun(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		tty        bool
		wantCode   ExitCode
		wantStdout string // substring
		wantStderr string // substring
	}{
		{name: "help", args: []string{"--help"}, wantCode: ExitOK, wantStdout: "CORE COMMANDS"},
		{name: "version", args: []string{"--version"}, wantCode: ExitOK, wantStdout: "tool version"},
		{name: "list piped", args: []string{"item", "list"}, wantCode: ExitOK, wantStdout: "3\tWrite docs\topen\t"},
		{name: "flag error shows usage on stderr", args: []string{"item", "list", "--limit", "0"},
			wantCode: ExitError, wantStderr: "invalid value for --limit: 0\n\nUsage:"},
		{name: "unknown flag is a usage error", args: []string{"item", "list", "--nope"},
			wantCode: ExitError, wantStderr: "unknown flag: --nope"},
		{name: "typo suggestion for nested command", args: []string{"item", "lsit"},
			wantCode: ExitError, wantStderr: "Did you mean this?\n\tlist"},
		{name: "non-interactive delete needs --yes", args: []string{"item", "delete", "3"},
			wantCode: ExitError, wantStderr: "--yes required when not running interactively"},
		{name: "empty result is success", args: []string{"item", "list", "--state", "closed", "--limit", "1", "--json", "id", "--jq", ".[] | select(.id > 99)"},
			wantCode: ExitOK},
		{name: "bare --json lists fields", args: []string{"item", "list", "--json"},
			wantCode: ExitError, wantStderr: "Specify one or more comma-separated fields for `--json`:\n  id\n  state"},
		{name: "noun without verb shows its help", args: []string{"item"}, wantCode: ExitOK, wantStdout: "AVAILABLE"},
		{name: "unknown root command suggests", args: []string{"itme"}, wantCode: ExitError, wantStderr: "Did you mean this?\n\titem"},
		{name: "help topic", args: []string{"help", "environment"}, wantCode: ExitOK, wantStdout: "TOOL_CONFIG_DIR"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TOOL_CONFIG_DIR", t.TempDir())
			ios, _, stdout, stderr := iostreams.Test()
			ios.SetStdoutTTY(tt.tty)
			code := Run(tt.args, ios)
			if code != tt.wantCode {
				t.Errorf("exit code = %d, want %d\nstderr: %s", code, tt.wantCode, stderr)
			}
			if !strings.Contains(stdout.String(), tt.wantStdout) {
				t.Errorf("stdout %q does not contain %q", stdout.String(), tt.wantStdout)
			}
			if !strings.Contains(stderr.String(), tt.wantStderr) {
				t.Errorf("stderr %q does not contain %q", stderr.String(), tt.wantStderr)
			}
		})
	}
}

func TestRun_promptsOnStderr(t *testing.T) {
	t.Setenv("TOOL_CONFIG_DIR", t.TempDir())
	ios, stdin, stdout, stderr := iostreams.Test()
	ios.SetStdinTTY(true)
	ios.SetStderrTTY(true) // stdout stays redirected
	stdin.WriteString("nope\n")

	code := Run([]string{"item", "delete", "3"}, ios)
	if code != ExitError {
		t.Errorf("exit = %d, want %d (mismatched confirmation)", code, ExitError)
	}
	if !strings.Contains(stderr.String(), "Type 3 to confirm deletion") {
		t.Errorf("prompt not on stderr: %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout must stay clean, got %q", stdout.String())
	}
}

func TestRun_noInputFlag(t *testing.T) {
	t.Setenv("TOOL_CONFIG_DIR", t.TempDir())
	ios, _, _, stderr := iostreams.Test()
	ios.SetStdinTTY(true)
	ios.SetStdoutTTY(true)
	ios.SetStderrTTY(true)

	code := Run([]string{"item", "delete", "3", "--no-input"}, ios)
	if code != ExitError || !strings.Contains(stderr.String(), "--yes required") {
		t.Errorf("exit = %d, stderr = %q; want --yes required", code, stderr.String())
	}
}

func TestRun_eofAtPromptIsCancel(t *testing.T) {
	t.Setenv("TOOL_CONFIG_DIR", t.TempDir())
	ios, _, _, _ := iostreams.Test()
	ios.SetStdinTTY(true)
	ios.SetStderrTTY(true)

	if code := Run([]string{"item", "delete", "3"}, ios); code != ExitCancel {
		t.Errorf("exit = %d, want %d: EOF at a prompt is never consent", code, ExitCancel)
	}
}

func TestSignalExitCode(t *testing.T) {
	for sig, want := range map[os.Signal]ExitCode{os.Interrupt: 130, syscall.SIGTERM: 143} {
		if got := signalExitCode(sig); got != want {
			t.Errorf("signalExitCode(%v) = %d, want %d", sig, got, want)
		}
	}
}
