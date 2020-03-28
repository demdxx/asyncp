package asyncp

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMuxErrorPanic(t *testing.T) {
	isError := false
	isPanic := false
	isFailover := false
	mux := NewTaskMux(
		WithPanicHandler(func(Task, Event, interface{}) { isPanic = true }),
		WithErrorHandler(func(Task, Event, error) { isError = true }),
	)
	mux.Handle(`error`, FuncTask(func(context.Context, Event, ResponseWriter) error { return fmt.Errorf(`test`) }))
	mux.Handle(`panic`, FuncTask(func(context.Context, Event, ResponseWriter) error { panic("test") }))
	mux.Failver(FuncTask(func(context.Context, Event, ResponseWriter) error { isFailover = true; return nil }))
	mux.Receive(mustMessageFrom(WithPayload(`error`, `test`)))
	mux.Receive(mustMessageFrom(WithPayload(`panic`, `test`)))
	mux.Receive(mustMessageFrom(WithPayload(`failover`, `test`)))

	assert.True(t, isError, `error`)
	assert.True(t, isPanic, `panic`)
	assert.True(t, isFailover, `failover`)
}
