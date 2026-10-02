package shared

import (
	"errors"
	"fmt"
	"testing"

	"example.com/tool/internal/api"
	"example.com/tool/pkg/cmdutil"
)

func TestParseIDArg(t *testing.T) {
	if id, err := ParseIDArg("#3"); err != nil || id != 3 {
		t.Fatalf("ParseIDArg(#3) = %v, %v", id, err)
	}
	_, err := ParseIDArg("x")
	var fe *cmdutil.FlagError
	if !errors.As(err, &fe) {
		t.Fatalf("ParseIDArg(x) error = %v, want FlagError", err)
	}
}

func TestNotFoundHint(t *testing.T) {
	err := NotFoundHint(fmt.Errorf("delete: %w", api.NotFoundError{ID: 7}))
	var nf api.NotFoundError
	if !errors.As(err, &nf) || nf.ID != 7 {
		t.Fatalf("hint lost the typed error: %v", err)
	}
	other := errors.New("boom")
	if NotFoundHint(other) != other {
		t.Fatal("other errors must pass through unchanged")
	}
}
