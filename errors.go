package asyncp

import "errors"

func errorString(err error) string {
	if err == nil {
		return ``
	}
	return err.Error()
}

func stringError(msg string) error {
	if msg == `` {
		return nil
	}
	return errors.New(msg)
}
