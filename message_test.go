package asyncp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMessageFrom(t *testing.T) {
	m, err := messageFrom([]byte(`raw`))
	assert.NoError(t, err)
	assert.Equal(t, []byte(`raw`), m.Body())

	m, err = messageFrom(`text`)
	assert.NoError(t, err)
	assert.Equal(t, []byte(`text`), m.Body())

	m, err = messageFrom(map[string]int{"a": 1})
	assert.NoError(t, err)
	assert.Equal(t, `{"a":1}`, string(m.Body()))

	_, err = messageFrom(nil)
	assert.ErrorIs(t, err, ErrNullMessageValue)
}

func TestMessageFromMarshalError(t *testing.T) {
	_, err := messageFrom(make(chan int))
	assert.Error(t, err)
}

func TestMustMessageFrom(t *testing.T) {
	assert.NotPanics(t, func() { mustMessageFrom(`ok`) })
	assert.Panics(t, func() { mustMessageFrom(nil) })
}

func TestMessageAccessors(t *testing.T) {
	m := message(`data`)
	assert.Equal(t, ``, m.ID())
	assert.NotNil(t, m.Context())
	assert.Equal(t, []byte(`data`), m.Body())
	assert.NoError(t, m.Ack())
}
