package asyncp

import (
	"encoding/json"
	"reflect"
)

// Payload represents interface of working with input data
type Payload interface {
	// Encode payload data to the bytes
	Encode() ([]byte, error)

	// Decode payload data into the target
	Decode(target any) error
}

func newPayload(val any) (Payload, error) {
	switch b := (val).(type) {
	case nil:
	case []byte:
		return dataPayload{bytes: b}, nil
	default:
		return &valuePayload{value: val}, nil
	}
	return dataPayload{}, nil
}

type dataPayload struct {
	bytes []byte
}

func (p dataPayload) Decode(target any) error {
	return json.Unmarshal(p.bytes, target)
}

func (p dataPayload) Encode() ([]byte, error) {
	return p.bytes, nil
}

func (p dataPayload) MarshalJSON() ([]byte, error) {
	return p.Encode()
}

type valuePayload struct {
	value any
}

func (p *valuePayload) Decode(target any) error {
	dst := valueFinal(reflect.ValueOf(target))
	src := valueFinal(reflect.ValueOf(p.value))
	dst.Set(src)
	return nil
}

func (p *valuePayload) Encode() ([]byte, error) {
	return json.Marshal(p.value)
}

func (p *valuePayload) MarshalJSON() ([]byte, error) {
	return p.Encode()
}

func valueFinal(v reflect.Value) reflect.Value {
	for v.IsValid() && (v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface) && !v.IsNil() {
		v = v.Elem()
	}
	return v
}
