package delete

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"example.com/tool/internal/api"
	"example.com/tool/internal/prompter"
	"example.com/tool/pkg/cmdutil"
	"example.com/tool/pkg/iostreams"
)

type DeleteOptions struct {
	IO        *iostreams.IOStreams
	APIClient func() (api.Client, error)
	Prompter  prompter.Prompter
	Ctx       context.Context

	ID        int
	Confirmed bool
}

func NewCmdDelete(f *cmdutil.Factory, runF func(*DeleteOptions) error) *cobra.Command {
	opts := &DeleteOptions{IO: f.IOStreams, APIClient: f.APIClient, Prompter: f.Prompter}

	cmd := &cobra.Command{
		Use:     "delete <id>",
		Aliases: []string{"rm"},
		Short:   "Delete an item",
		Long: `Delete an item.

Deletion is permanent. You are asked to type the item ID to confirm; pass --yes to
skip the confirmation, which is required when not running interactively.`,
		Example: `$ tool item delete 3
$ tool item delete 3 --yes`,
		Args: cmdutil.ExactArgs(1, "cannot delete item: id argument required"),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Ctx = cmd.Context()
			id, err := strconv.Atoi(args[0])
			if err != nil || id < 1 {
				return cmdutil.FlagErrorf("invalid item id: %q", args[0])
			}
			opts.ID = id
			if !opts.IO.CanPrompt() && !opts.Confirmed {
				return cmdutil.FlagErrorf("--yes required when not running interactively")
			}
			if runF != nil {
				return runF(opts)
			}
			return deleteRun(opts)
		},
	}
	cmd.Flags().BoolVarP(&opts.Confirmed, "yes", "y", false, "Confirm deletion without prompting")
	return cmd
}

func deleteRun(opts *DeleteOptions) error {
	client, err := opts.APIClient()
	if err != nil {
		return err
	}
	ctx := opts.Ctx
	if ctx == nil {
		ctx = context.Background()
	}

	if !opts.Confirmed {
		if err := opts.Prompter.ConfirmDeletion(strconv.Itoa(opts.ID)); err != nil {
			if errors.Is(err, prompter.ErrInterrupt) {
				return cmdutil.CancelError
			}
			return err
		}
	}

	if err := client.DeleteItem(ctx, opts.ID); err != nil {
		var nf api.NotFoundError
		if errors.As(err, &nf) {
			return fmt.Errorf("%w\nrun `tool item list --state all` to see existing items", err)
		}
		return err
	}

	if opts.IO.IsStdoutTTY() {
		cs := opts.IO.ColorScheme()
		fmt.Fprintf(opts.IO.ErrOut, "%s Deleted item %d\n", cs.SuccessIcon(), opts.ID)
	}
	return nil
}
