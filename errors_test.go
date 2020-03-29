package asyncp

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorConversion(t *testing.T) {
	assert.Nil(t, stringError(``))
	assert.Empty(t, errorString(nil))
	assert.NotNil(t, stringError(`not nil`))
	assert.Equal(t, `test error`, errorString(fmt.Errorf(`test error`)))
}
