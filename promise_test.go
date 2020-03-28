package asyncp

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testTask = FuncTask(func(ctx context.Context, event Event, responseWriter ResponseWriter) error {
	fmt.Println(event.Name())
	return nil
})

func TestPromise(t *testing.T) {
	mux := NewTaskMux()
	pr1 := newPoromise(mux, nil, "test", testTask)
	pr2 := pr1.Then(testTask)
	pr3 := pr2.Then(testTask)

	assert.Equal(t, "test.1", pr1.TargetEventName())
	assert.Equal(t, "test.1", pr2.EventName())
	assert.Equal(t, "test.2", pr2.TargetEventName())
	assert.Equal(t, "test.2", pr3.EventName())
	assert.Equal(t, "" /* */, pr3.TargetEventName())
}
