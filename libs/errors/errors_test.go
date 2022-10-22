package errors

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorConversion(t *testing.T) {
	assert.Nil(t, StringError(``))
	assert.Empty(t, ErrorString(nil))
	assert.NotNil(t, StringError(`not nil`))
	assert.Equal(t, `test error`, ErrorString(fmt.Errorf(`test error`)))
}
