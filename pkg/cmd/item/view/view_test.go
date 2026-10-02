package view

import (
	"bytes"
	"context"
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
	return &api.MemoryClient{Items: []api.Item{{
		ID: 3, Title: "Write docs", State: "open", URL: "https://example.com/items/3",
		UpdatedAt: fixedNow.Add(-2 * time.Hour),
	}}}, nil
}

// fakeBrowser records URLs instead of launching a program.
type fakeBrowser struct{ urls []string }

func (b *fakeBrowser) Browse(_ context.Context, url string) error {
	b.urls = append(b.urls, url)
	return nil
}

// Layer 1: parsing and validation only.
func TestNewCmdView(t *testing.T) {
	tests := []struct {
		name    string
		cli     string
		wantErr string
		wantID  api.ItemID
		wantWeb bool
	}{
		{name: "id", cli: "3", wantID: 3},
		{name: "hash id", cli: "#3", wantID: 3},
		{name: "web", cli: "3 --web", wantID: 3, wantWeb: true},
		{name: "missing id", cli: "", wantErr: "cannot view item: id argument required"},
		{name: "bad id", cli: "abc", wantErr: `invalid item id: "abc"`},
		{name: "web and json", cli: "3 --web --json title", wantErr: "specify only one of `--web` or `--json`"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios, _, _, _ := iostreams.Test()
			f := &cmdutil.Factory{IOStreams: ios}

			var got *ViewOptions
			cmd := NewCmdView(f, func(o *ViewOptions) error { got = o; return nil })
			cmd.SetArgs(strings.Fields(tt.cli))
			cmd.SetIn(&bytes.Buffer{})
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return cmdutil.FlagErrorWrap(err) })

			_, err := cmd.ExecuteC()
			if tt.wantErr != "" {
				var fe *cmdutil.FlagError
				if err == nil || err.Error() != tt.wantErr || !errors.As(err, &fe) {
					t.Fatalf("error = %v, want FlagError %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.ID != tt.wantID || got.Web != tt.wantWeb {
				t.Errorf("got id=%d web=%v", got.ID, got.Web)
			}
		})
	}
}

// Layer 2: behavior and exact output in both TTY modes.
func TestViewRun(t *testing.T) {
	tests := []struct {
		name       string
		tty        bool
		web        bool
		id         api.ItemID
		wantStdout string
		wantStderr string
		wantURLs   []string
		wantErr    string
	}{
		{name: "tty", tty: true, id: 3,
			wantStdout: "Write docs #3\nopen • updated about 2 hours ago\nhttps://example.com/items/3\n"},
		{name: "piped", tty: false, id: 3,
			wantStdout: "id\t3\ntitle\tWrite docs\nstate\topen\nupdated\t2024-01-02T10:00:00Z\nurl\thttps://example.com/items/3\n"},
		{name: "web on tty", tty: true, web: true, id: 3,
			wantStderr: "Opening https://example.com/items/3 in your browser.\n",
			wantURLs:   []string{"https://example.com/items/3"}},
		{name: "web piped is quiet", tty: false, web: true, id: 3,
			wantURLs: []string{"https://example.com/items/3"}},
		{name: "not found", tty: false, id: 99,
			wantErr: "item 99 not found\nrun `tool item list --state all` to see existing items"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios, _, stdout, stderr := iostreams.Test()
			ios.SetStdoutTTY(tt.tty)
			ios.SetStderrTTY(tt.tty)
			ios.SetColorEnabled(false)
			b := &fakeBrowser{}

			err := viewRun(&ViewOptions{
				IO: ios, APIClient: fixtureClient, Browser: b,
				Now: func() time.Time { return fixedNow },
				ID:  tt.id, Web: tt.web,
			})
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("error = %v, want %q", err, tt.wantErr)
				}
				var nf api.NotFoundError
				if !errors.As(err, &nf) {
					t.Fatalf("error %v does not wrap api.NotFoundError", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if stdout.String() != tt.wantStdout {
				t.Errorf("stdout = %q, want %q", stdout.String(), tt.wantStdout)
			}
			if stderr.String() != tt.wantStderr {
				t.Errorf("stderr = %q, want %q", stderr.String(), tt.wantStderr)
			}
			if strings.Join(b.urls, " ") != strings.Join(tt.wantURLs, " ") {
				t.Errorf("browsed %q, want %q", b.urls, tt.wantURLs)
			}
		})
	}
}
