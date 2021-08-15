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
		testTask = func(_ context.Context, event Event, rw ResponseWriter) error {
			var i int
			if err := event.Payload().Decode(&i); err != nil {
				return err
			}
			if i >= 2 {
				return rw.WriteResonse(i + 1)
			}
			return rw.RepeatWithResponse(i + 1)
		}
		failoverTask = func(_ context.Context, event Event, rw ResponseWriter) error {
			return event.Payload().Decode(&result)
		}
		mux = NewTaskMux(
			WithMainExecContext(ctx),
			WithResponseFactory(NewProxyResponseFactory()),
		)
	)
	_ = mux.Failver(failoverTask)
	mux.Handle("test", testTask)

	err := mux.ExecuteEvent(event)
	assert.NoError(t, err)
	assert.Equal(t, int(3), result)
}

func TestExtFuncTask(t *testing.T) {
	type item struct {
		Text string `json:"text"`
	}
	var (
		res    = ""
		ctx    = context.Background()
		event1 = WithPayload("test1", item{Text: "test1"})
		event2 = WithPayload("test2", item{Text: "test2"})
		mux    = NewTaskMux(
			WithMainExecContext(ctx),
			WithResponseFactory(NewProxyResponseFactory()),
		)
	)

	mux.Handle("test1", func(it *item) (*item, error) {
		res = it.Text
		return nil, nil
	})

	mux.Handle("test2", func(ctx context.Context, it *item, event Event, rw ResponseWriter) (*item, error) {
		res = it.Text
		return it, nil
	})

	err := mux.ExecuteEvent(event1)
	assert.NoError(t, err)
	assert.Equal(t, "test1", res)

	err = mux.ExecuteEvent(event2)
	assert.NoError(t, err)
	assert.Equal(t, "test2", res)
}
