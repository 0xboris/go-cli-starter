package cmdutil

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// StringEnumFlag defines a string flag restricted to options: it validates the
// value, shows `{a|b|c}` in help, and registers shell completion — in one call.
func StringEnumFlag(cmd *cobra.Command, p *string, name, shorthand, defaultValue string, options []string, usage string) {
	*p = defaultValue
	val := &enumValue{string: p, options: options}
	cmd.Flags().VarP(val, name, shorthand, fmt.Sprintf("%s: {%s}", usage, strings.Join(options, "|")))
	_ = cmd.RegisterFlagCompletionFunc(name, func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return options, cobra.ShellCompDirectiveNoFileComp
	})
}

type enumValue struct {
	string  *string
	options []string
}

func (e *enumValue) Set(value string) error {
	for _, o := range e.options {
		if strings.EqualFold(o, value) {
			*e.string = o
			return nil
		}
	}
	return fmt.Errorf("valid values are {%s}", strings.Join(e.options, "|"))
}

func (e *enumValue) String() string { return *e.string }
func (e *enumValue) Type() string   { return "string" }

// DisableAuthCheck marks a command as not requiring authentication.
func DisableAuthCheck(cmd *cobra.Command) {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations["skipAuthCheck"] = "true"
}

// ExactArgs is cobra.ExactArgs with a FlagError and a helpful message.
func ExactArgs(n int, msg string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) > n {
			return FlagErrorf("too many arguments")
		}
		if len(args) < n {
			return FlagErrorf("%s", msg)
		}
		return nil
	}
}

// NoArgsQuoteReminder rejects positional args and hints at unquoted values,
// the most common cause of stray arguments (`--title my title`).
func NoArgsQuoteReminder(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return nil
	}
	errMsg := fmt.Sprintf("unknown argument %q", args[0])
	if len(args) > 1 {
		errMsg = fmt.Sprintf("unknown arguments %q", args)
	}
	hasValueFlag := false
	cmd.Flags().Visit(func(f *pflagFlag) {
		if f.Value.Type() != "bool" {
			hasValueFlag = true
		}
	})
	if hasValueFlag {
		errMsg += "; please quote all values that have spaces"
	}
	return FlagErrorf("%s", errMsg)
}
