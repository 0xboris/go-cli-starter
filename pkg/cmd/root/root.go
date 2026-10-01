package root

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	itemCmd "example.com/tool/pkg/cmd/item"
	versionCmd "example.com/tool/pkg/cmd/version"
	"example.com/tool/pkg/cmdutil"
)

func NewCmdRoot(f *cmdutil.Factory, version, buildDate string) *cobra.Command {
	failed = false
	cmd := &cobra.Command{
		Use:   "tool <command> <subcommand> [flags]",
		Short: "Tool CLI",
		Long:  "Work with items from the command line.",
		Example: strings.TrimSpace(`
$ tool item list
$ tool item list --state all --json id,title
$ tool item delete 3 --yes`),
		Version: versionCmd.Format(version, buildDate),
		// Main prints errors exactly once and shows usage only for FlagError.
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	cmd.SetVersionTemplate("{{.Version}}")
	cmd.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
		if err == pflag.ErrHelp {
			return err
		}
		return cmdutil.FlagErrorWrap(err)
	})
	cmd.PersistentFlags().Bool("help", false, "Show help for command")
	cmd.Flags().BoolP("version", "v", false, "Show tool version")
	cmd.SetHelpFunc(helpFunc)
	cmd.SetUsageFunc(usageFunc)
	cmd.CompletionOptions.HiddenDefaultCmd = false

	cmd.AddGroup(&cobra.Group{ID: "core", Title: "Core commands"})

	cmd.AddCommand(itemCmd.NewCmdItem(f))
	cmd.AddCommand(versionCmd.NewCmdVersion(f, version, buildDate))

	for _, topic := range helpTopics {
		cmd.AddCommand(newHelpTopic(topic))
	}

	return cmd
}

// helpFunc renders gh-style sections. Unknown nested subcommands get suggestions
// and mark the run as failed (help funcs cannot return errors).
func helpFunc(cmd *cobra.Command, args []string) {
	out := cmd.OutOrStdout()
	if cmd.Annotations["helpTopic"] == "true" {
		fmt.Fprintln(out, cmd.Long)
		return
	}
	// Help funcs receive the full argv. Anything left after the command path that
	// isn't a flag is an unknown subcommand ("tool item lsit").
	if cmd.HasSubCommands() && !cmd.Flags().Changed("help") {
		depth := 0
		for c := cmd; c.HasParent(); c = c.Parent() {
			depth++
		}
		var positional []string
		for _, a := range args {
			if !strings.HasPrefix(a, "-") {
				positional = append(positional, a)
			}
		}
		if len(positional) > depth {
			unknown := positional[depth]
			if cmd.SuggestionsMinimumDistance <= 0 {
				cmd.SuggestionsMinimumDistance = 2 // cobra only suggests at the root by default
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "unknown command %q for %q\n", unknown, cmd.CommandPath())
			if s := cmd.SuggestionsFor(unknown); len(s) > 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "\nDid you mean this?\n\t%s\n", strings.Join(s, "\n\t"))
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "\nRun '%s --help' for usage.\n", cmd.CommandPath())
			failed = true
			return
		}
	}

	section := func(title, body string) {
		if strings.TrimSpace(body) == "" {
			return
		}
		fmt.Fprintf(out, "%s\n", strings.ToUpper(title))
		for _, line := range strings.Split(strings.TrimRight(body, "\n"), "\n") {
			fmt.Fprintf(out, "  %s\n", line)
		}
		fmt.Fprintln(out)
	}

	desc := cmd.Long
	if desc == "" {
		desc = cmd.Short
	}
	fmt.Fprintf(out, "%s\n\n", desc)
	section("Usage", cmd.UseLine())
	if len(cmd.Aliases) > 0 {
		section("Aliases", strings.Join(cmd.Aliases, ", "))
	}
	for _, g := range cmd.Groups() {
		section(g.Title, commandList(cmd, g.ID))
	}
	if cmd.HasParent() {
		section("Available commands", commandList(cmd, ""))
	} else {
		section("Additional commands", commandList(cmd, ""))
	}
	if !cmd.HasParent() {
		section("Help topics", topicList(cmd))
	}
	section("Flags", cmd.NonInheritedFlags().FlagUsages())
	section("Inherited flags", cmd.InheritedFlags().FlagUsages())
	if f := cmd.Annotations["help:json-fields"]; f != "" {
		section("JSON fields", strings.ReplaceAll(f, ",", ", "))
	}
	section("Arguments", cmd.Annotations["help:arguments"])
	section("Environment variables", cmd.Annotations["help:environment"])
	section("Examples", cmd.Example)
	section("Learn more", "Use `tool <command> <subcommand> --help` for more information about a command.\n"+
		"Learn about exit codes using `tool help exit-codes`")
}

func commandList(cmd *cobra.Command, groupID string) string {
	var b strings.Builder
	for _, c := range cmd.Commands() {
		if c.GroupID != groupID || !c.IsAvailableCommand() || c.Annotations["helpTopic"] == "true" {
			continue
		}
		fmt.Fprintf(&b, "%-12s %s\n", c.Name()+":", c.Short)
	}
	return b.String()
}

func topicList(cmd *cobra.Command) string {
	var b strings.Builder
	for _, c := range cmd.Commands() {
		if c.Annotations["helpTopic"] == "true" {
			fmt.Fprintf(&b, "%-12s %s\n", c.Name()+":", c.Short)
		}
	}
	return b.String()
}

var failed bool

// HasFailed reports whether help output represented an error (e.g. unknown subcommand).
func HasFailed() bool { return failed }

// usageFunc is the terse usage printed after a FlagError (to stderr): the usage
// line, the flags, and where to get full help.
func usageFunc(cmd *cobra.Command) error {
	out := cmd.ErrOrStderr()
	fmt.Fprintf(out, "Usage:  %s\n", cmd.UseLine())
	if cmd.HasAvailableSubCommands() {
		fmt.Fprintln(out, "\nAvailable commands:")
		for _, c := range cmd.Commands() {
			if c.IsAvailableCommand() && c.Annotations["helpTopic"] != "true" {
				fmt.Fprintf(out, "  %s\n", c.Name())
			}
		}
	}
	if flags := cmd.LocalFlags().FlagUsages(); flags != "" {
		fmt.Fprintf(out, "\nFlags:\n%s", flags)
	}
	fmt.Fprintf(out, "\nRun '%s --help' for more information.\n", cmd.CommandPath())
	return nil
}
