// Package build holds values injected at link time:
//
//	go build -ldflags "-X example.com/tool/internal/build.Version=v1.2.3 -X example.com/tool/internal/build.Date=2024-01-15"
package build

import "runtime/debug"

// Version is the semantic version of this build ("DEV" for local builds).
var Version = "DEV"

// Date is the build date in YYYY-MM-DD format.
var Date = ""

func init() {
	// `go install example.com/tool/cmd/tool@vX.Y.Z` builds carry the module version.
	if Version == "DEV" {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "(devel)" && info.Main.Version != "" {
			Version = info.Main.Version
		}
	}
}
