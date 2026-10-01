// Package acceptance runs txtar scripts against the CLI in-process.
// For tools that hit real services, put these behind a build tag (gh uses
// `//go:build acceptance`) and run them separately.
package acceptance

import (
	"os"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"

	"example.com/tool/internal/app"
)

func TestMain(m *testing.M) {
	os.Exit(testscript.RunMain(m, map[string]func() int{
		"tool": func() int { return int(app.Main()) },
	}))
}

func TestScripts(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "testdata",
		Setup: func(env *testscript.Env) error {
			env.Setenv("HOME", env.WorkDir) // sandbox: never touch the real home dir
			env.Setenv("TOOL_CONFIG_DIR", env.WorkDir+"/.config/tool")
			return nil
		},
		RequireExplicitExec: true,
	})
}
