package root

import (
	"strings"

	"github.com/spf13/cobra"
)

type helpTopic struct {
	name  string
	short string
	long  string
}

// Help topics are hidden commands rendered as pages: `tool help environment`.
// Keep `environment` complete: every env var the tool reads, with precedence.
var helpTopics = []helpTopic{
	{
		name:  "environment",
		short: "Environment variables that can be used with tool",
		long: strings.TrimSpace(`
TOOL_CONFIG_DIR: the directory where tool stores configuration files. If not set,
$XDG_CONFIG_HOME/tool or ~/.config/tool is used.

TOOL_PAGER, PAGER (in order of precedence): a terminal paging program to send
standard output to, e.g. "less". Set to "cat" to disable.

TOOL_PROMPT_DISABLED: set to any value to disable interactive prompting in the terminal
(same as the --no-input flag). Commands that need input then fail with the flags to use.

TOOL_FORCE_TTY: set to any value to force terminal-style output even when the output
is redirected. A numeric value sets the output width in columns.

TOOL_DEBUG: set to a truthy value to enable verbose output on standard error.

TOOL_BROWSER, BROWSER (in order of precedence): the program, with optional arguments
separated by spaces, used to open URLs (e.g. by "item view --web"). Defaults to
"open" on macOS, "xdg-open" on Linux and "rundll32 url.dll,FileProtocolHandler" on Windows.

NO_COLOR: set to any value to avoid printing ANSI escape sequences for color output.

CLICOLOR: set to "0" to disable printing ANSI colors in output.

TERM: when set to "dumb", colors and spinners are disabled.

CLICOLOR_FORCE: set to a value other than "0" to keep ANSI colors in output even when
the output is piped.`),
	},
	{
		name:  "exit-codes",
		short: "Exit codes used by tool",
		long: strings.TrimSpace(`
tool follows normal conventions regarding exit codes.

- If a command completes successfully, the exit code will be 0

- If a command fails for any reason, the exit code will be 1

- If a command is running but gets cancelled, the exit code will be 2

- If a command requires authentication, the exit code will be 4

- If a command is interrupted by a signal, the exit code will be 128 plus the signal
  number: 130 for Ctrl-C (SIGINT), 143 for SIGTERM. Ctrl-C at a prompt counts as a
  cancellation (exit code 2)

NOTE: It is possible that a particular command may have more exit codes, so it is a
good practice to check documentation for the command if you are relying on exit codes
to control some behavior.`),
	},
	{
		name:  "formatting",
		short: "Formatting options for JSON data exported from tool",
		long: strings.TrimSpace(`
Some tool commands support exporting the data as JSON as an alternative to their usual
line-based plain text output. This is suitable for passing structured data to scripts.
The JSON output is enabled with the --json option, followed by the list of fields to fetch.
Use the flag without a value to get the list of available fields.

The --jq option accepts a query in jq syntax and will print only the resulting values
that match the query. Strings are printed raw.

The --template (-t) flag accepts a Go template that is executed against the JSON data.

EXAMPLES
  $ tool item list --json id,title
  $ tool item list --json id,title --jq '.[] | select(.id > 1) | .title'
  $ tool item list --json id,title --template '{{range .}}{{.id}}: {{.title}}{{"\n"}}{{end}}'`),
	},
}

func newHelpTopic(t helpTopic) *cobra.Command {
	return &cobra.Command{
		Use:         t.name,
		Short:       t.short,
		Long:        t.long,
		Hidden:      true,
		Annotations: map[string]string{"helpTopic": "true"},
		Args:        cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := cmd.OutOrStdout().Write([]byte(t.long + "\n"))
			return err
		},
	}
}
