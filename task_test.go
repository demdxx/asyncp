package asyncp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTask(t *testing.T) {
	var (
		result   int
		ctx      = context.Background()
		event    = WithPayload("test", int(1))
		testTask = FuncTask(func(_ context.Context, event Event, rw ResponseWriter) error {
			var i int
			if err := event.Payload().Decode(&i); err != nil {
				return err
			}
			if i == 2 {
				return rw.WriteResonse(i + 1)
			}
			return rw.RepeatWithResponse(i + 1)
		})
		failoverTask = FuncTask(func(_ context.Context, event Event, rw ResponseWriter) error {
			return event.Payload().Decode(&result)
		})
		mux = NewTaskMux(
			WithMainExecContext(ctx),
			WithResponseFactory(NewProxyResponseFactory()),
		)
	)
	mux.Failver(failoverTask)
	mux.Handle("test", testTask)

	err := mux.ExecuteEvent(event)
	assert.NoError(t, err)
	assert.Equal(t, int(3), result)
}
