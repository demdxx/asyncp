package asyncp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultAllocator(t *testing.T) {
	a := newDefaultEventAllocator()
	e, err := a.Decode(mustMessageFrom(struct{ S string }{S: "message"}))
	assert.NoError(t, err)
	if assert.NotNil(t, e) {
		assert.NoError(t, a.Release(e))
	}
}
