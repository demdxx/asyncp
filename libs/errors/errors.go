package errors

import "errors"

var (
	// ErrSkipEvent in case of repeat count exceeds the limit
	ErrSkipEvent = errors.New("skip event")

	// ErrNil in case of empty response
	ErrNil = errors.New("nil response")
)

func ErrorString(err error) string {
	if err == nil {
		return ``
	}
	return err.Error()
}

func StringError(msg string) error {
	if msg == `` {
		return nil
	}
	return errors.New(msg)
}

func Is(err, target error) bool {
	return errors.Is(err, target)
}
