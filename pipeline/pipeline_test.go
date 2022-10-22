package pipeline

import (
	"context"
	"testing"

	"github.com/demdxx/asyncp/v2"
	"github.com/stretchr/testify/assert"
)

type testItem struct {
	Iterations int64
	Index      int64
	Value      int64
	Status     string
}

func TestPipeline(t *testing.T) {
	var (
		ctx          = context.Background()
		totalSuccess = 0
		event        = asyncp.WithPayload(`sum`, &testItem{Iterations: 10})
		pipe         = New(
			`sum`, asyncp.FuncTask(func(ctx context.Context, event asyncp.Event, responseWriter asyncp.ResponseWriter) error {
				var (
					sum  int64
					data testItem
				)
				_ = event.Payload().Decode(&data)
				for i := int64(0); i < data.Iterations; i++ {
					sum += i
					if err := responseWriter.WriteResonse(&testItem{Index: i, Value: sum}); err != nil {
						return err
					}
				}
				return nil
			}),
			`result`, asyncp.FuncTask(func(ctx context.Context, event asyncp.Event, responseWriter asyncp.ResponseWriter) error {
				data := new(testItem)
				_ = event.Payload().Decode(data)
				data.Status = "success"
				return responseWriter.WriteResonse(data)
			}),
		)
	)

	err := pipe.Execute(ctx, event, asyncp.ResponseHandlerFnk(func(payload any) error {
		if payload.(*testItem).Status == "success" {
			totalSuccess++
		}
		return nil
	}))

	assert.NoError(t, err)
	assert.Equal(t, 10, totalSuccess)
}
