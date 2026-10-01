package cmdutil

import (
	"example.com/tool/internal/api"
	"example.com/tool/internal/config"
	"example.com/tool/internal/prompter"
	"example.com/tool/pkg/iostreams"
)

// Factory carries every dependency a command may need. Cheap, always-needed
// values are eager; anything that costs I/O or can fail is a lazy func so that
// `--help`, `version` and completion work offline with a broken config.
// Commands copy only what they use into their Options struct.
type Factory struct {
	AppVersion     string
	ExecutablePath string

	IOStreams *iostreams.IOStreams
	Prompter  prompter.Prompter

	Config    func() (*config.Config, error)
	APIClient func() (api.Client, error)
}
