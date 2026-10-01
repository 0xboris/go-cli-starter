// Package app is the real entrypoint. Main returns an exit code instead of calling
// os.Exit so deferred cleanup runs and tests can invoke the CLI in-process.
package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"example.com/tool/internal/api"
	"example.com/tool/internal/build"
	"example.com/tool/internal/config"
	"example.com/tool/internal/prompter"
	"example.com/tool/pkg/cmd/root"
	"example.com/tool/pkg/cmdutil"
	"example.com/tool/pkg/iostreams"
)

type ExitCode int

const (
	ExitOK     ExitCode = 0
	ExitError  ExitCode = 1
	ExitCancel ExitCode = 2
	ExitAuth   ExitCode = 4
)

// AuthError means the command requires authentication (exit 4).
type AuthError struct{ err error }

func (e *AuthError) Error() string { return e.err.Error() }

func Main() ExitCode {
	return Run(os.Args[1:], iostreams.System())
}

// Run executes the CLI with args on the given streams. Tests and testscript use it.
func Run(args []string, ios *iostreams.IOStreams) ExitCode {
	stderr := ios.ErrOut

	f := newFactory(ios)
	applyEnvAndConfig(f) // env > config > default

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	rootCmd := root.NewCmdRoot(f, build.Version, build.Date)
	rootCmd.SetArgs(args)
	rootCmd.SetIn(ios.In)
	rootCmd.SetOut(ios.Out)
	rootCmd.SetErr(ios.ErrOut)

	cmd, err := rootCmd.ExecuteContextC(ctx)
	code := exitCodeFor(err, cmd, ios, debugEnabled())
	if code == ExitOK && root.HasFailed() {
		code = ExitError
	}
	if ctx.Err() != nil && code == ExitError {
		fmt.Fprintln(stderr) // keep the shell prompt on its own line after ^C
		code = ExitCancel
	}
	return code
}

func exitCodeFor(err error, cmd *cobra.Command, ios *iostreams.IOStreams, debug bool) ExitCode {
	stderr := ios.ErrOut
	var (
		authErr   *AuthError
		pagerErr  *iostreams.ErrClosedPagerPipe
		noResults cmdutil.NoResultsError
	)
	switch {
	case err == nil:
		return ExitOK
	case errors.Is(err, cmdutil.SilentError):
		return ExitError
	case cmdutil.IsUserCancellation(err):
		fmt.Fprintln(stderr)
		return ExitCancel
	case errors.As(err, &authErr):
		fmt.Fprintln(stderr, authErr.Error())
		return ExitAuth
	case errors.As(err, &pagerErr):
		return ExitOK // the user quit the pager
	case errors.As(err, &noResults):
		if ios.IsStdoutTTY() {
			fmt.Fprintln(stderr, noResults.Error())
		}
		return ExitOK // empty is not a failure
	}
	printError(stderr, err, cmd, debug)
	return ExitError
}

func printError(out io.Writer, err error, cmd *cobra.Command, debug bool) {
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		fmt.Fprintf(out, "error connecting to %s\n", dnsErr.Name)
		if debug {
			fmt.Fprintln(out, dnsErr)
		}
		fmt.Fprintln(out, "check your internet connection")
		return
	}

	fmt.Fprintln(out, err)

	var flagErr *cmdutil.FlagError
	if cmd != nil && (errors.As(err, &flagErr) || strings.HasPrefix(err.Error(), "unknown command ")) {
		if !strings.HasSuffix(err.Error(), "\n") {
			fmt.Fprintln(out)
		}
		_ = cmd.Usage() // terse usage to stderr (see root.usageFunc)
	}
}

func newFactory(ios *iostreams.IOStreams) *cmdutil.Factory {
	exe, _ := os.Executable()
	f := &cmdutil.Factory{
		AppVersion:     build.Version,
		ExecutablePath: exe,
		IOStreams:      ios,
		Prompter:       prompter.New(ios.In, ios.Out),
	}

	var cachedCfg *config.Config
	var cfgErr error
	f.Config = func() (*config.Config, error) { // lazy + cached
		if cachedCfg == nil && cfgErr == nil {
			cachedCfg, cfgErr = config.Load()
		}
		return cachedCfg, cfgErr
	}

	// Replace with your real client; keep it lazy so --help never touches the network.
	f.APIClient = func() (api.Client, error) {
		now := time.Now()
		return &api.MemoryClient{Items: []api.Item{
			{ID: 3, Title: "Write docs", State: "open", UpdatedAt: now.Add(-2 * time.Hour)},
			{ID: 2, Title: "Add --json output", State: "open", UpdatedAt: now.Add(-26 * time.Hour)},
			{ID: 1, Title: "Initial release", State: "closed", UpdatedAt: now.Add(-40 * 24 * time.Hour)},
		}}, nil
	}
	return f
}

// applyEnvAndConfig wires I/O behavior with explicit precedence: env > config > default.
// A broken config is a warning here; commands that need config fail lazily.
func applyEnvAndConfig(f *cmdutil.Factory) {
	ios := f.IOStreams
	cfg, err := f.Config()
	if err != nil {
		fmt.Fprintf(ios.ErrOut, "warning: %v\n", err)
		cfg = config.NewFromMap(map[string]string{})
	}

	if os.Getenv(config.EnvPromptDisabled) != "" || cfg.GetOrDefault("prompt") == "disabled" {
		ios.SetNeverPrompt(true)
	}

	pager := os.Getenv(config.EnvPager)
	if pager == "" {
		pager = cfg.GetOrDefault("pager")
	}
	if pager == "" {
		pager = os.Getenv("PAGER")
	}
	ios.SetPager(pager)
}

func debugEnabled() bool {
	v := os.Getenv(config.EnvDebug)
	return v != "" && v != "0" && v != "false"
}
