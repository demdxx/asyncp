package asyncp

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorConversion(t *testing.T) {
	assert.Nil(t, stringError(``))
	assert.Empty(t, errorString(nil))
	assert.NotNil(t, stringError(`not nil`))
	assert.Equal(t, `test error`, errorString(fmt.Errorf(`test error`)))
}

func TestMultierror(t *testing.T) {
	var err multiError
	for i := 0; i < 100; i++ {
		err.Add(fmt.Errorf(`Error %d`, i))
	}
	assert.Equal(t, 100, len(strings.Split(err.Error(), "\n")))
}
