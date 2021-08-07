package asyncp

import (
	"context"
	"testing"
	"time"

	"github.com/demdxx/asyncp/monitor"
	"github.com/demdxx/asyncp/monitor/kvstorage"
	"github.com/stretchr/testify/assert"
)

type dummyMetric struct{}

func (dummyMetric) RegisterApplication(appInfo *monitor.ApplicationInfo) error        { return nil }
func (dummyMetric) DeregisterApplication() error                                      { return nil }
func (dummyMetric) ReceiveEvent(event monitor.EventType) error                        { return nil }
func (dummyMetric) ExecuteTask(event monitor.EventType, execTime time.Duration) error { return nil }
func (dummyMetric) ExecuteFailoverTask(_ monitor.EventType, _ time.Duration) error    { return nil }

func TestOptions(t *testing.T) {
	var (
		options   Options
		optionFnk = []Option{
			WithMainExecContext(context.Background()),
			WithPanicHandler(func(Task, Event, interface{}) {}),
			WithErrorHandler(func(Task, Event, error) {}),
			WithContextWrapper(func(ctx context.Context) context.Context { return ctx }),
			WithStreamResponseMap("item1", &testPublisher{name: "test-"}),
			WithResponseFactory(NewMultistreamResponseFactory("item2", &testPublisher{name: "test2"})),
			WithStreamResponsePublisher(&testPublisher{name: "test3"}),
			WithMonitorDefaults("test", dummyMetric{}),
			WithCluster("test", ClusterWithReader(&kvstorage.ClusterInfoReader{})),
			WithEventAllocator(newDefaultEventAllocator()),
		}
	)
	for _, opt := range optionFnk {
		opt(&options)
	}
	assert.NotNil(t, options.MainExecContext)
	assert.NotNil(t, options.PanicHandler)
	assert.NotNil(t, options.ErrorHandler)
	assert.NotNil(t, options.ContextWrapper)
	assert.NotNil(t, options.ResponseFactory)
	assert.NotNil(t, options.Cluster)
	assert.NotNil(t, options.EventAllocator)
	assert.NotNil(t, options._eventAllocator())
	assert.NotNil(t, (&Options{})._eventAllocator())
}
