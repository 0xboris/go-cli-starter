package list

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"example.com/tool/internal/api"
	"example.com/tool/internal/tableprinter"
	"example.com/tool/pkg/cmdutil"
	"example.com/tool/pkg/iostreams"
)

type ListOptions struct {
	IO        *iostreams.IOStreams
	APIClient func() (api.Client, error)
	Exporter  cmdutil.Exporter
	Now       func() time.Time
	Ctx       context.Context

	State string
	Limit int
}

func NewCmdList(f *cmdutil.Factory, runF func(*ListOptions) error) *cobra.Command {
	opts := &ListOptions{IO: f.IOStreams, APIClient: f.APIClient, Now: time.Now}

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List items",
		Long:    "List items, newest first. By default only open items are shown.",
		Example: `# List open items
$ tool item list

# List every item as JSON
$ tool item list --state all --json id,title,state`,
		Args: cmdutil.NoArgsQuoteReminder,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Ctx = cmd.Context()
			if opts.Limit < 1 {
				return cmdutil.FlagErrorf("invalid value for --limit: %v", opts.Limit)
			}
			if runF != nil {
				return runF(opts)
			}
			return listRun(opts)
		},
	}

	cmdutil.StringEnumFlag(cmd, &opts.State, "state", "s", "open", []string{"open", "closed", "all"}, "Filter by state")
	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", 30, "Maximum number of items to fetch")
	cmdutil.AddJSONFlags(cmd, &opts.Exporter, api.ItemFields)
	return cmd
}

func listRun(opts *ListOptions) error {
	client, err := opts.APIClient()
	if err != nil {
		return err
	}
	ctx := opts.Ctx
	if ctx == nil {
		ctx = context.Background()
	}

	var items []api.Item
	err = opts.IO.RunWithProgress("Fetching items", func() error {
		items, err = client.ListItems(ctx, api.ListParams{State: opts.State, Limit: opts.Limit})
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to list items: %w", err)
	}

	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IO, items) // [] for empty, never an error
	}
	if len(items) == 0 {
		return cmdutil.NewNoResultsError(fmt.Sprintf("no %s items found", opts.State))
	}

	isTTY := opts.IO.IsStdoutTTY()
	cs := opts.IO.ColorScheme()
	if isTTY {
		fmt.Fprintf(opts.IO.Out, "\nShowing %d %s items\n\n", len(items), opts.State)
	}

	tp := tableprinter.New(opts.IO, "ID", "Title", "Updated")
	if !isTTY {
		tp = tableprinter.New(opts.IO) // piped: no header
	}
	for _, it := range items {
		if isTTY {
			tp.AddField("#"+strconv.Itoa(it.ID), stateColor(cs, it.State))
		} else {
			tp.AddField(strconv.Itoa(it.ID))
		}
		tp.AddField(it.Title)
		if !isTTY {
			tp.AddField(it.State) // color carries state on a TTY; spell it out when piped
		}
		tp.AddTimeField(opts.Now(), it.UpdatedAt, cs.Muted)
		tp.EndRow()
	}
	return tp.Render()
}

func stateColor(cs *iostreams.ColorScheme, state string) func(string) string {
	if state == "closed" {
		return cs.Red
	}
	return cs.Green
}
