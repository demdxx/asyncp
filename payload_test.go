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
