package asyncp

import (
	"encoding/json"
)

// Payload represents interface of working with input data
type Payload interface {
	// Encode payload data to the bytes
	Encode() ([]byte, error)

	// Decode payload data into the target
	Decode(target interface{}) error
}

type payload struct {
	bytes []byte
}

func newPayload(val interface{}) (payload, error) {
	var (
		data []byte
		err  error
	)
	switch b := (val).(type) {
	case nil:
	case []byte:
		data = b
	default:
		data, err = json.Marshal(b)
	}
	return payload{bytes: data}, err
}

func (p payload) Decode(target interface{}) error {
	return json.Unmarshal(p.bytes, target)
}

func (p payload) Encode() ([]byte, error) {
	return p.bytes, nil
}
