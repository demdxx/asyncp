package asyncp

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEventMethods(t *testing.T) {
	event := WithPayload(`test`, 100)

	errEvent := event.WithError(fmt.Errorf(`test`))
	assert.NotNil(t, errEvent.Err(), `error`)

	event.SetComplete(true)
	assert.True(t, event.IsComplete())

	event2 := event.WithName(`test2`)
	assert.Equal(t, `test2`, event2.Name())

	assert.NotEqual(t, ``, event2.String())
}

func TestEventAfter(t *testing.T) {
	event1 := WithPayload(`test1`, 100)
	event2 := WithPayload(`test2`, 100).After(event1)
	event3 := WithPayload(`test3`, 100).After(event2)

	assert.ElementsMatch(t, []string{`test1`, `test2`}, event3.DoneTasks())
}
