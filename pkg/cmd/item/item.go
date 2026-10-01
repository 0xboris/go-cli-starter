// Package item is the sample noun. Copy its shape for real nouns, then delete it.
package item

import (
	"github.com/spf13/cobra"

	deleteCmd "example.com/tool/pkg/cmd/item/delete"
	listCmd "example.com/tool/pkg/cmd/item/list"
	"example.com/tool/pkg/cmdutil"
)

func NewCmdItem(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "item <command>",
		Short:   "Manage items",
		Long:    "Work with items.",
		GroupID: "core",
		Annotations: map[string]string{
			"help:arguments": "An item can be supplied as an argument by its numeric ID.",
		},
	}
	cmd.AddCommand(listCmd.NewCmdList(f, nil))
	cmd.AddCommand(deleteCmd.NewCmdDelete(f, nil))
	return cmd
}
