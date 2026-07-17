package asyncp

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMuxErrorPanic(t *testing.T) {
	var (
		isError    = false
		isPanic    = false
		isFailover = false
		lastEvent  Event
		mux        = NewTaskMux(
			WithPanicHandler(func(Task, Event, any) { isPanic = true }),
			WithErrorHandler(func(Task, Event, error) { isError = true }),
			WithStreamResponseMap(&testPublisher{name: "test"}),
		)
	)
	_ = mux.Handle(`error`, FuncTask(func(_ context.Context, e Event, _ ResponseWriter) error {
		lastEvent = e.(*event).Copy()
		return fmt.Errorf(`test`)
	}))
	_ = mux.Handle(`panic`, FuncTask(func(context.Context, Event, ResponseWriter) error { panic("test") }))
	_ = mux.Handle(`panic>noop`, FuncTask(func(context.Context, Event, ResponseWriter) error { panic("noop") }))
	_ = mux.Failover(FuncTask(func(context.Context, Event, ResponseWriter) error { isFailover = true; return nil }))
	_ = mux.Receive(mustMessageFrom(WithPayload(`error`, `test`)))
	_ = mux.Receive(mustMessageFrom(WithPayload(`panic`, `test`)))
	_ = mux.Receive(mustMessageFrom(WithPayload(`failover`, `test`)))

	assert.True(t, isError, `error`)
	assert.True(t, isPanic, `panic`)
	assert.True(t, isFailover, `failover`)
	assert.NoError(t, mux.Close())
	assert.Equal(t, map[string][]string{
		`error`: {},
		`panic`: {"noop"},
		`noop`:  {},
	}, mux.TaskMap())
	totalTasks, completeTasks := mux.CompleteTasks(lastEvent)
	assert.ElementsMatch(t, []string{`error`}, totalTasks)
	assert.ElementsMatch(t, []string{}, completeTasks)
}

func TestMuxFinishInitWithoutCluster(t *testing.T) {
	mux := NewTaskMux()
	assert.NoError(t, mux.FinishInit())
}

func TestMuxExecuteEventUnknownWithoutFailover(t *testing.T) {
	mux := NewTaskMux()
	err := mux.ExecuteEvent(WithPayload(`unknown`, nil))
	assert.NoError(t, err)
}

func TestMuxReceiveDecodeError(t *testing.T) {
	mux := NewTaskMux()
	err := mux.Receive(message(`not-json`))
	assert.Error(t, err)
}

func TestMuxHandleDuplicateTaskPanics(t *testing.T) {
	mux := NewTaskMux()
	_ = mux.Handle(`dup`, testTask)
	assert.Panics(t, func() { mux.Handle(`dup`, testTask) })
}

func TestMuxBorrowResponseWriterDefault(t *testing.T) {
	mux := NewTaskMux()
	var executed bool
	promise := mux.Handle(`test`, testTask)
	promise.Then(FuncTask(func(context.Context, Event, ResponseWriter) error {
		executed = true
		return nil
	}))

	wr := mux.borrowResponseWriter(context.Background(), promise, WithPayload(`test`, nil))
	assert.NoError(t, wr.WriteResonse(`hello`))
	assert.True(t, executed)
}

func TestMuxCloseClosesTasks(t *testing.T) {
	mux := NewTaskMux()
	ct := &closeableTask{FuncTask: testTask}
	_ = mux.Handle(`test`, ct)

	assert.NoError(t, mux.Close())
	assert.True(t, ct.closed)
}

func TestMuxCloseNilMux(t *testing.T) {
	var mux *TaskMux
	assert.NoError(t, mux.Close())
}
