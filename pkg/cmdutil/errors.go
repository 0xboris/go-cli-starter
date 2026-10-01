package cmdutil

import (
	"errors"
	"fmt"

	"example.com/tool/internal/prompter"
)

// SilentError means the command already reported the problem; exit 1 without printing.
var SilentError = errors.New("SilentError")

// CancelError signals that the user cancelled the operation; exit 2, no message.
var CancelError = errors.New("CancelError")

// IsUserCancellation reports whether err represents a user cancellation.
func IsUserCancellation(err error) bool {
	return errors.Is(err, CancelError) || errors.Is(err, prompter.ErrInterrupt)
}

// FlagError is a usage error: the message is followed by the command's usage.
type FlagError struct {
	err error
}

func (fe *FlagError) Error() string { return fe.err.Error() }
func (fe *FlagError) Unwrap() error { return fe.err }

// FlagErrorf returns a new FlagError that wraps an error produced by fmt.Errorf.
func FlagErrorf(format string, args ...any) error {
	return FlagErrorWrap(fmt.Errorf(format, args...))
}

// FlagErrorWrap wraps err (for example a pflag parse error) as a FlagError.
func FlagErrorWrap(err error) error { return &FlagError{err: err} }

// NoResultsError is not a failure: the CLI exits 0 and prints the message only on a TTY.
type NoResultsError struct {
	message string
}

func (e NoResultsError) Error() string { return e.message }

// NewNoResultsError returns a NoResultsError with the given message.
func NewNoResultsError(message string) NoResultsError {
	return NoResultsError{message: message}
}

// MutuallyExclusive returns a FlagError with message if more than one condition is true.
func MutuallyExclusive(message string, conditions ...bool) error {
	n := 0
	for _, c := range conditions {
		if c {
			n++
		}
	}
	if n > 1 {
		return FlagErrorf("%s", message)
	}
	return nil
}
