package delete

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"example.com/tool/internal/api"
	"example.com/tool/internal/prompter"
	"example.com/tool/pkg/cmdutil"
	"example.com/tool/pkg/iostreams"
)

func TestNewCmdDelete(t *testing.T) {
	tests := []struct {
		name    string
		tty     bool
		cli     string
		wantErr string
		wantID  api.ItemID
		wantYes bool
	}{
		{name: "tty without --yes prompts later", tty: true, cli: "3", wantID: 3},
		{name: "non-tty requires --yes", tty: false, cli: "3", wantErr: "--yes required when not running interactively"},
		{name: "non-tty with --yes", tty: false, cli: "3 --yes", wantID: 3, wantYes: true},
		{name: "missing id", tty: true, cli: "", wantErr: "cannot delete item: id argument required"},
		{name: "bad id", tty: true, cli: "abc", wantErr: `invalid item id: "abc"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios, _, _, _ := iostreams.Test()
			ios.SetStdinTTY(tt.tty)
			ios.SetStdoutTTY(tt.tty)
			ios.SetStderrTTY(tt.tty)
			f := &cmdutil.Factory{IOStreams: ios}

			var got *DeleteOptions
			cmd := NewCmdDelete(f, func(o *DeleteOptions) error { got = o; return nil })
			cmd.SetArgs(strings.Fields(tt.cli))
			cmd.SetIn(&bytes.Buffer{})
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)

			_, err := cmd.ExecuteC()
			if tt.wantErr != "" {
				var fe *cmdutil.FlagError
				if err == nil || err.Error() != tt.wantErr || !errors.As(err, &fe) {
					t.Fatalf("error = %v, want FlagError %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ID != tt.wantID || got.Confirmed != tt.wantYes {
				t.Errorf("got id=%d yes=%v", got.ID, got.Confirmed)
			}
		})
	}
}

// Prompts are drawn on stderr, so prompting depends on stdin and stderr, not stdout.
func TestNewCmdDelete_promptChannel(t *testing.T) {
	tests := []struct {
		name                  string
		stdin, stdout, stderr bool
		neverPrompt           bool
		wantErr               bool
	}{
		{name: "stdout redirected still prompts", stdin: true, stdout: false, stderr: true},
		{name: "stderr redirected cannot prompt", stdin: true, stdout: true, stderr: false, wantErr: true},
		{name: "stdin piped cannot prompt", stdin: false, stdout: true, stderr: true, wantErr: true},
		{name: "--no-input on a full TTY", stdin: true, stdout: true, stderr: true, neverPrompt: true, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios, _, _, _ := iostreams.Test()
			ios.SetStdinTTY(tt.stdin)
			ios.SetStdoutTTY(tt.stdout)
			ios.SetStderrTTY(tt.stderr)
			ios.SetNeverPrompt(tt.neverPrompt)
			cmd := NewCmdDelete(&cmdutil.Factory{IOStreams: ios}, func(*DeleteOptions) error { return nil })
			cmd.SetArgs([]string{"3"})
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			_, err := cmd.ExecuteC()
			if tt.wantErr != (err != nil) {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && err.Error() != "--yes required when not running interactively" {
				t.Errorf("unexpected message: %v", err)
			}
		})
	}
}

func TestDeleteRun(t *testing.T) {
	tests := []struct {
		name       string
		confirmed  bool
		prompt     func(string) error
		wantErr    error
		wantStderr string
		wantLeft   int
	}{
		{
			name:       "confirmed by typing the id",
			prompt:     func(v string) error { return nil },
			wantStderr: "✓ Deleted item 1\n",
			wantLeft:   0,
		},
		{
			name:     "prompt interrupted is a cancellation",
			prompt:   func(string) error { return prompter.ErrInterrupt },
			wantErr:  cmdutil.CancelError,
			wantLeft: 1,
		},
		{
			name:       "--yes skips the prompt",
			confirmed:  true,
			wantStderr: "✓ Deleted item 1\n",
			wantLeft:   0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios, _, stdout, stderr := iostreams.Test()
			ios.SetStdoutTTY(true)
			client := &api.MemoryClient{Items: []api.Item{{ID: 1, Title: "x", State: "open"}}}
			pm := &prompter.Mock{} // unset funcs panic: unexpected prompts fail the test
			if tt.prompt != nil {
				pm.ConfirmDeletionFunc = func(v string) error {
					if v != "1" {
						t.Errorf("prompted for %q, want %q", v, "1")
					}
					return tt.prompt(v)
				}
			}

			err := deleteRun(&DeleteOptions{
				IO:        ios,
				APIClient: func() (api.Client, error) { return client, nil },
				Prompter:  pm,
				ID:        1,
				Confirmed: tt.confirmed,
			})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if stdout.Len() != 0 {
				t.Errorf("delete must not write to stdout, got %q", stdout.String())
			}
			if stderr.String() != tt.wantStderr {
				t.Errorf("stderr = %q, want %q", stderr.String(), tt.wantStderr)
			}
			if len(client.Items) != tt.wantLeft {
				t.Errorf("items left = %d, want %d", len(client.Items), tt.wantLeft)
			}
		})
	}
}
