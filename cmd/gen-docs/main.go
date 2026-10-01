// gen-docs renders man pages and markdown from the real command tree, so docs
// can never drift from flags. CI runs it and fails on `git diff`.
//
//	go run ./cmd/gen-docs --man-page --doc-path share/man/man1
//	go run ./cmd/gen-docs --website  --doc-path docs/commands
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/spf13/cobra/doc"

	"example.com/tool/internal/build"
	"example.com/tool/pkg/cmd/root"
	"example.com/tool/pkg/cmdutil"
	"example.com/tool/pkg/iostreams"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("gen-docs", flag.ContinueOnError)
	manPage := flags.Bool("man-page", false, "Generate manual pages")
	website := flags.Bool("website", false, "Generate markdown for the website")
	dir := flags.String("doc-path", "", "Path to the output directory")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *dir == "" || (!*manPage && !*website) {
		return fmt.Errorf("usage: gen-docs {--man-page|--website} --doc-path <dir>")
	}

	ios, _, _, _ := iostreams.Test()
	// Stub factory: docs generation must not need config, credentials or network.
	rootCmd := root.NewCmdRoot(&cmdutil.Factory{IOStreams: ios}, build.Version, build.Date)
	rootCmd.DisableAutoGenTag = true

	if err := os.MkdirAll(*dir, 0o755); err != nil {
		return err
	}
	if *website {
		if err := doc.GenMarkdownTree(rootCmd, *dir); err != nil {
			return err
		}
	}
	if *manPage {
		header := &doc.GenManHeader{Title: "TOOL", Section: "1", Source: "tool " + build.Version, Manual: "Tool CLI manual"}
		if err := doc.GenManTree(rootCmd, header, *dir); err != nil {
			return err
		}
	}
	return nil
}
