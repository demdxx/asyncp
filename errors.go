package asyncp

import (
	"bytes"
	"errors"
)

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

type multiError []error

func (e multiError) Error() string {
	if len(e) == 0 {
		return ``
	}
	var buff bytes.Buffer
	for i, err := range e {
		if i > 0 {
			buff.WriteByte('\n')
		}
		buff.Write([]byte(`- `))
		buff.WriteString(err.Error())
	}
	return buff.String()
}

func (e *multiError) Add(err error) {
	if e != nil && err != nil {
		*e = append(*e, err)
	}
}

func (e multiError) AsError() error {
	if len(e) == 0 {
		return nil
	} else if len(e) == 1 {
		return e[0]
	}
	return e
}
