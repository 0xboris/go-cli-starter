// Package archtest enforces the layering rules from quality-cli's layers.md with
// plain `go test`, so they hold even where golangci-lint (depguard) isn't run.
package archtest

import (
	"os/exec"
	"strings"
	"testing"
)

const module = "example.com/tool"

// Packages below the command layer, and what they must never depend on, directly
// or transitively. Add your domain and adapter packages here.
var rules = []struct {
	pkg       string
	forbidden []string
}{
	{module + "/internal/api/...", belowCommands},
	{module + "/internal/browser/...", belowCommands},
	{module + "/internal/config/...", belowCommands},
	{module + "/internal/build/...", belowCommands},
	{module + "/internal/text/...", belowCommands},
}

var belowCommands = []string{
	"github.com/spf13/cobra",
	"github.com/spf13/pflag",
	module + "/pkg/", // commands, cmdutil, iostreams
	module + "/internal/app",
}

func TestLayering(t *testing.T) {
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go toolchain not on PATH")
	}
	for _, r := range rules {
		out, err := exec.Command(goBin, "list", "-deps", r.pkg).CombinedOutput()
		if err != nil {
			t.Fatalf("go list -deps %s: %v\n%s", r.pkg, err, out)
		}
		for _, dep := range strings.Fields(string(out)) {
			for _, bad := range r.forbidden {
				if dep == bad || strings.HasPrefix(dep, bad) {
					t.Errorf("%s depends on %s (imports must point down; see layers.md)", r.pkg, dep)
				}
			}
		}
	}
}
