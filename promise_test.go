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

func TestPromiseTargetEvent(t *testing.T) {
	mux := NewTaskMux()
	pr1 := mux.Handle("test", testTask)
	_ = mux.Handle("other", testTask)

	returned := pr1.TargetEvent("other")
	assert.Same(t, pr1, returned)
	assert.ElementsMatch(t, []string{"other"}, pr1.TargetEventName())
}

type closeableTask struct {
	FuncTask
	closed bool
}

func (t *closeableTask) Close() error {
	t.closed = true
	return nil
}

func TestPromiseClose(t *testing.T) {
	mux := NewTaskMux()
	ct := &closeableTask{FuncTask: testTask}
	p := mux.Handle("test", ct).(*promise)

	assert.NoError(t, p.Close())
	assert.True(t, ct.closed)
}

func TestPromiseCloseWithoutCloser(t *testing.T) {
	mux := NewTaskMux()
	p := mux.Handle("test", testTask).(*promise)
	assert.NoError(t, p.Close())
}
