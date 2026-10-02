package version

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"example.com/tool/pkg/cmdutil"
)

func NewCmdVersion(f *cmdutil.Factory, version, buildDate string) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "version",
		Short:  "Show tool version",
		Hidden: true, // `tool --version` is the documented form; both work
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprint(f.IOStreams.Out, Format(version, buildDate))
			return err
		},
	}
	cmdutil.DisableAuthCheck(cmd)
	return cmd
}

// Format renders "tool version 1.2.3 (2024-01-15)\n<release url>\n".
func Format(version, buildDate string) string {
	version = strings.TrimPrefix(version, "v")
	dateStr := ""
	if buildDate != "" {
		dateStr = fmt.Sprintf(" (%s)", buildDate)
	}
	return fmt.Sprintf("tool version %s%s\n%s\n", version, dateStr, changelogURL(version))
}

func changelogURL(version string) string {
	if !semver.MatchString(version) { // DEV builds, commit hashes, pre-releases
		return "https://example.com/tool/releases/latest"
	}
	return fmt.Sprintf("https://example.com/tool/releases/tag/v%s", version)
}

var semver = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
