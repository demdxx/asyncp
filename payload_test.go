package asyncp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPayload(t *testing.T) {
	pl, err := newPayload("test")
	assert.NoError(t, err)

	var s string
	err = pl.Decode(&s)
	assert.NoError(t, err, `decode`)
	assert.Equal(t, `test`, s)
}

func TestPayloadNil(t *testing.T) {
	// A nil value falls through to an empty dataPayload{} rather than a nil
	// Payload interface value.
	pl, err := newPayload(nil)
	assert.NoError(t, err)
	if assert.NotNil(t, pl) {
		data, err := pl.Encode()
		assert.NoError(t, err)
		assert.Empty(t, data)
	}
}

func TestDataPayload(t *testing.T) {
	pl, err := newPayload([]byte(`{"a":1}`))
	assert.NoError(t, err)

	data, err := pl.Encode()
	assert.NoError(t, err)
	assert.Equal(t, `{"a":1}`, string(data))

	marshalled, err := pl.(dataPayload).MarshalJSON()
	assert.NoError(t, err)
	assert.Equal(t, data, marshalled)

	var m map[string]int
	assert.NoError(t, pl.Decode(&m))
	assert.Equal(t, 1, m["a"])
}

func TestValuePayloadMarshalJSON(t *testing.T) {
	pl, err := newPayload(42)
	assert.NoError(t, err)

	data, err := pl.(*valuePayload).MarshalJSON()
	assert.NoError(t, err)
	assert.Equal(t, `42`, string(data))
}

func TestValuePayloadDecodePointer(t *testing.T) {
	type payloadStruct struct{ A int }
	pl, err := newPayload(&payloadStruct{A: 5})
	assert.NoError(t, err)

	var dst payloadStruct
	assert.NoError(t, pl.Decode(&dst))
	assert.Equal(t, 5, dst.A)
}
