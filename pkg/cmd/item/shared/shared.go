// Package shared holds code several item verbs need. It is the place for finders,
// listers and display helpers; a verb never imports another verb's package.
package shared

import (
	"errors"
	"fmt"

	"example.com/tool/internal/api"
	"example.com/tool/pkg/cmdutil"
	"example.com/tool/pkg/iostreams"
)

// ParseIDArg turns a positional argument into an ItemID. Bad input is a usage
// error (FlagError), so the user sees the usage text.
func ParseIDArg(arg string) (api.ItemID, error) {
	id, err := api.ParseItemID(arg)
	if err != nil {
		return 0, cmdutil.FlagErrorWrap(err)
	}
	return id, nil
}

// NotFoundHint adds the next step to a domain NotFoundError; other errors pass through.
// Domain errors state facts; commands add what the user should do.
func NotFoundHint(err error) error {
	var nf api.NotFoundError
	if errors.As(err, &nf) {
		return fmt.Errorf("%w\nrun `tool item list --state all` to see existing items", err)
	}
	return err
}

// StateColor colors a state on a TTY; when piped, callers print the state as text.
func StateColor(cs *iostreams.ColorScheme, state string) func(string) string {
	if state == "closed" {
		return cs.Red
	}
	return cs.Green
}
