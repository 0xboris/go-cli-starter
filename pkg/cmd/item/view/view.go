package view

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"example.com/tool/internal/api"
	"example.com/tool/internal/browser"
	"example.com/tool/internal/tableprinter"
	"example.com/tool/internal/text"
	"example.com/tool/pkg/cmd/item/shared"
	"example.com/tool/pkg/cmdutil"
	"example.com/tool/pkg/iostreams"
)

type ViewOptions struct {
	IO        *iostreams.IOStreams
	APIClient func() (api.Client, error)
	Browser   browser.Browser
	Exporter  cmdutil.Exporter
	Now       func() time.Time
	Ctx       context.Context

	ID  api.ItemID
	Web bool
}

func NewCmdView(f *cmdutil.Factory, runF func(*ViewOptions) error) *cobra.Command {
	opts := &ViewOptions{IO: f.IOStreams, APIClient: f.APIClient, Browser: f.Browser, Now: time.Now}

	cmd := &cobra.Command{
		Use:   "view <id>",
		Short: "View an item",
		Long: `Display the title, state, last update and URL of an item.

With --web, open the item in the web browser instead (see TOOL_BROWSER in
'tool help environment').`,
		Example: `$ tool item view 3
$ tool item view 3 --web
$ tool item view 3 --json title,url --jq .url`,
		Args: cmdutil.ExactArgs(1, "cannot view item: id argument required"),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Ctx = cmd.Context()
			id, err := shared.ParseIDArg(args[0])
			if err != nil {
				return err
			}
			opts.ID = id
			if err := cmdutil.MutuallyExclusive("specify only one of `--web` or `--json`",
				opts.Web, opts.Exporter != nil); err != nil {
				return err
			}
			if runF != nil {
				return runF(opts)
			}
			return viewRun(opts)
		},
	}
	cmd.Flags().BoolVarP(&opts.Web, "web", "w", false, "Open the item in the browser")
	cmdutil.AddJSONFlags(cmd, &opts.Exporter, api.ItemFields)
	return cmd
}

// viewRun orchestrates: domain call → browser adapter, exporter, or rendering.
func viewRun(opts *ViewOptions) error {
	ctx := opts.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := opts.APIClient()
	if err != nil {
		return err
	}
	item, err := client.GetItem(ctx, opts.ID)
	if err != nil {
		return shared.NotFoundHint(err)
	}

	if opts.Web {
		if opts.IO.IsStderrTTY() {
			_, _ = fmt.Fprintf(opts.IO.ErrOut, "Opening %s in your browser.\n", item.URL)
		}
		return opts.Browser.Browse(ctx, item.URL)
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IO, item)
	}
	if opts.IO.IsStdoutTTY() {
		return printHuman(opts.IO, item, opts.Now())
	}
	return printRaw(opts.IO, item, opts.Now())
}

func printHuman(io *iostreams.IOStreams, it *api.Item, now time.Time) error {
	cs := io.ColorScheme()
	state := shared.StateColor(cs, it.State)(it.State)
	_, err := fmt.Fprintf(io.Out, "%s %s\n%s • updated %s\n%s\n",
		cs.Bold(it.Title), cs.Muted("#"+it.ID.String()),
		state, text.FuzzyAgo(now, it.UpdatedAt),
		it.URL)
	return err
}

// printRaw writes one "key<TAB>value" line per field: no color, absolute time,
// escaped fields (the tableprinter's non-TTY contract).
func printRaw(io *iostreams.IOStreams, it *api.Item, now time.Time) error {
	tp := tableprinter.New(io)
	row := func(k, v string) { tp.AddField(k); tp.AddField(v); tp.EndRow() }
	row("id", it.ID.String())
	row("title", it.Title)
	row("state", it.State)
	tp.AddField("updated")
	tp.AddTimeField(now, it.UpdatedAt)
	tp.EndRow()
	row("url", it.URL)
	return tp.Render()
}
