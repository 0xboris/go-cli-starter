package list

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"example.com/tool/internal/api"
	"example.com/tool/pkg/cmdutil"
	"example.com/tool/pkg/iostreams"
)

var fixedNow = time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC)

func fixtureClient() (api.Client, error) {
	return &api.MemoryClient{Items: []api.Item{
		{ID: 2, Title: "Second", State: "open", UpdatedAt: fixedNow.Add(-3 * time.Hour)},
		{ID: 1, Title: "First", State: "closed", UpdatedAt: fixedNow.Add(-48 * time.Hour)},
	}}, nil
}

// Layer 1: flag parsing and validation, no I/O (runF captures the options).
func TestNewCmdList(t *testing.T) {
	tests := []struct {
		name      string
		cli       string
		wantErr   string
		wantState string
		wantLimit int
		wantJSON  []string
	}{
		{name: "defaults", cli: "", wantState: "open", wantLimit: 30},
		{name: "state and limit", cli: "--state all -L 5", wantState: "all", wantLimit: 5},
		{name: "invalid limit", cli: "--limit 0", wantErr: "invalid value for --limit: 0"},
		{name: "invalid state", cli: "--state nope", wantErr: `invalid argument "nope" for "-s, --state" flag: valid values are {open|closed|all}`},
		{name: "json fields", cli: "--json id,title", wantState: "open", wantLimit: 30, wantJSON: []string{"id", "title"}},
		{name: "unknown json field", cli: "--json nope", wantErr: "Unknown JSON field: \"nope\""},
		{name: "jq without json", cli: "--jq .", wantErr: "cannot use `--jq` without specifying `--json`"},
		{name: "stray argument", cli: "--state all oops", wantErr: `unknown argument "oops"; please quote all values that have spaces`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios, _, _, _ := iostreams.Test()
			f := &cmdutil.Factory{IOStreams: ios}

			var got *ListOptions
			cmd := NewCmdList(f, func(o *ListOptions) error { got = o; return nil })
			cmd.SetArgs(strings.Fields(tt.cli))
			cmd.SetIn(&bytes.Buffer{})
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return cmdutil.FlagErrorWrap(err) })

			_, err := cmd.ExecuteC()
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.State != tt.wantState || got.Limit != tt.wantLimit {
				t.Errorf("got state=%q limit=%d, want %q %d", got.State, got.Limit, tt.wantState, tt.wantLimit)
			}
			if tt.wantJSON != nil {
				if got.Exporter == nil || strings.Join(got.Exporter.Fields(), ",") != strings.Join(tt.wantJSON, ",") {
					t.Errorf("json fields = %v, want %v", got.Exporter, tt.wantJSON)
				}
			}
		})
	}
}

// Layer 2: behavior and exact output, in both TTY modes, from the same fixture.
func TestListRun(t *testing.T) {
	tests := []struct {
		name       string
		tty        bool
		state      string
		wantStdout string
	}{
		{
			name:  "tty",
			tty:   true,
			state: "all",
			wantStdout: "\nShowing 2 all items\n\n" +
				"ID  TITLE   UPDATED\n" +
				"#2  Second  about 3 hours ago\n" +
				"#1  First   about 2 days ago\n",
		},
		{
			name:  "piped: no header, tabs, explicit state, RFC3339",
			tty:   false,
			state: "all",
			wantStdout: "2\tSecond\topen\t2024-01-02T09:00:00Z\n" +
				"1\tFirst\tclosed\t2023-12-31T12:00:00Z\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios, _, stdout, stderr := iostreams.Test()
			ios.SetStdoutTTY(tt.tty)
			err := listRun(&ListOptions{IO: ios, APIClient: fixtureClient, Now: func() time.Time { return fixedNow }, State: tt.state, Limit: 30})
			if err != nil {
				t.Fatal(err)
			}
			if stdout.String() != tt.wantStdout {
				t.Errorf("stdout:\n%q\nwant:\n%q", stdout.String(), tt.wantStdout)
			}
			if stderr.Len() != 0 {
				t.Errorf("unexpected stderr: %q", stderr.String())
			}
		})
	}
}

func TestListRun_noResults(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	empty := func() (api.Client, error) { return &api.MemoryClient{}, nil }
	err := listRun(&ListOptions{IO: ios, APIClient: empty, Now: time.Now, State: "open", Limit: 30})
	var nr cmdutil.NoResultsError
	if !errors.As(err, &nr) {
		t.Fatalf("want NoResultsError, got %v", err)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout should be empty, got %q", stdout.String())
	}
}

func TestListRun_json(t *testing.T) {
	ios, _, stdout, _ := iostreams.Test()
	f := &cmdutil.Factory{IOStreams: ios, APIClient: fixtureClient}
	cmd := NewCmdList(f, func(o *ListOptions) error { o.Now = func() time.Time { return fixedNow }; return listRun(o) })
	cmd.SetArgs([]string{"--state", "all", "--json", "id,title", "--jq", ".[].title"})
	cmd.SetOut(io.Discard)
	if _, err := cmd.ExecuteC(); err != nil {
		t.Fatal(err)
	}
	if got, want := stdout.String(), "Second\nFirst\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
