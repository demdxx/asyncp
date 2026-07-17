package asyncp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResponseHandlerFnk(t *testing.T) {
	var received []any
	fnk := ResponseHandlerFnk(func(response any) error {
		received = append(received, response)
		return nil
	})

	assert.NoError(t, fnk.WriteResonse("write"))
	assert.NoError(t, fnk.RepeatWithResponse("repeat"))
	assert.NoError(t, fnk.Release())
	assert.Equal(t, []any{"write", "repeat"}, received)
}

func TestResponseWriterExecContextDefault(t *testing.T) {
	wr := &responseWriter{}
	assert.Equal(t, context.Background(), wr.execContext())
}
