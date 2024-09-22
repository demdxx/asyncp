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
	_ = mux.Failver(FuncTask(func(context.Context, Event, ResponseWriter) error { isFailover = true; return nil }))
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
