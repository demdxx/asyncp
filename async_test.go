package asyncp

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAsyncTask(t *testing.T) {
	var (
		wg           sync.WaitGroup
		recoverCount = int32(0)
		executeCount = int32(0)
		// Recover handler
		recf = func(rec any) {
			defer wg.Done()
			atomic.AddInt32(&recoverCount, 1)
		}
		// Main task handler
		task = FuncTask(func(ctx context.Context, event Event, rw ResponseWriter) error {
			var i int
			if err := event.Payload().Decode(&i); err != nil {
				return err
			}
			return rw.WriteResonse(i)
		})
		// Response write wrapper
		resp = ResponseHandlerFnk(func(response any) error {
			if response.(int)%2 != 0 {
				return fmt.Errorf(`test`)
			}
			defer wg.Done()
			atomic.AddInt32(&executeCount, 1)
			return nil
		})
		// Convert task to async
		atask = task.Async(WithWorkerCount(0), WithWorkerPoolSize(2), WithRecoverHandler(recf))
	)

	defer func() { _ = atask.Close() }()

	// Process all events asynchronously
	for i := 0; i < 100; i++ {
		wg.Add(1)
		err := atask.Execute(context.Background(), WithPayload(`test`, i), resp)
		assert.NoError(t, err)
	}

	wg.Wait()

	assert.Equal(t, int32(50), atomic.LoadInt32(&recoverCount), `recover count`)
	assert.Equal(t, int32(50), atomic.LoadInt32(&executeCount), `execute count`)
}
