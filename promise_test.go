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
	pr1 := mux.Handle("test", testTask)
	pr2 := pr1.Then(testTask)
	pr3 := pr2.Then(testTask)

	assert.ElementsMatch(t, []string{"test.1"}, pr1.TargetEventName())
	assert.Equal(t, "test", pr2.AfterEventName())
	assert.Equal(t, "test.1", pr2.EventName())
	assert.ElementsMatch(t, []string{"test.2"}, pr2.TargetEventName())
	assert.Equal(t, "test.1", pr3.AfterEventName())
	assert.Equal(t, "test.2", pr3.EventName())
	assert.Nil(t, pr3.TargetEventName())
}
