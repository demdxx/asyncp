package asyncp

import "github.com/demdxx/asyncp/v2/libs/errors"

var (
	// ErrSkipEvent in case of repeat count exceeds the limit
	ErrSkipEvent = errors.ErrSkipEvent

	// ErrNil in case of empty response
	ErrNil = errors.ErrNil
)

func errorString(err error) string {
	return errors.ErrorString(err)
}

func stringError(msg string) error {
	return errors.StringError(msg)
}
