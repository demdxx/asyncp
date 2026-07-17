package asyncp

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testPublisher struct{ name string }

func (p *testPublisher) String() string                                     { return p.name }
func (p *testPublisher) Publish(ctx context.Context, messages ...any) error { return nil }
func (p *testPublisher) PublishAndReturnIDs(ctx context.Context, messages ...any) ([]string, error) {
	return nil, nil
}

func TestMultistreamResponseFactory(t *testing.T) {
	fc := NewMultistreamResponseFactory(
		"item1", "item2", &testPublisher{name: "test1"},
		regexp.MustCompile(`item\d+`), &testPublisher{name: "test2"},
		&testPublisher{name: "test3"},
	).(*mutistreamResponseFactory)

	pub := fc.publisher(WithPayload("item2", nil))
	assert.Equal(t, "test1", pub.(fmt.Stringer).String())

	pub = fc.publisher(WithPayload("item3", nil))
	assert.Equal(t, "test2", pub.(fmt.Stringer).String())

	pub = fc.publisher(WithPayload("something", nil))
	assert.Equal(t, "test3", pub.(fmt.Stringer).String())

	assert.Panics(t, func() { NewMultistreamResponseFactory(100) })
}

func TestStreamResponseFactory(t *testing.T) {
	pub := &recordingPublisher{}
	fc := NewStreamResponseFactory(pub).(*streamResponseFactory)
	mux := NewTaskMux()
	fc.SetMux(mux)

	promise := mux.Handle("test", testTask)
	wr := fc.Borrow(context.Background(), promise, WithPayload("test", "value"))
	assert.NoError(t, wr.WriteResonse("hello"))
	assert.Len(t, pub.messages, 1)

	assert.NoError(t, wr.Release())
	// Releasing twice must be a no-op, not a double pool.Put panic/corruption.
	assert.NoError(t, wr.Release())
}

func TestProxyResponseFactory(t *testing.T) {
	fc := NewProxyResponseFactory().(*proxyResponseFactory)
	mux := NewTaskMux()
	fc.SetMux(mux)

	var executed bool
	promise := mux.Handle("test", testTask)
	promise.Then(FuncTask(func(context.Context, Event, ResponseWriter) error {
		executed = true
		return nil
	}))

	// Writing a response from the "test" promise must dispatch to its
	// chained target task ("test.1") through the mux, not just publish.
	wr := fc.Borrow(context.Background(), promise, WithPayload("test", "value"))
	assert.NoError(t, wr.WriteResonse("hello"))
	assert.True(t, executed)
	assert.NoError(t, wr.Release())
}

func TestStreamResponseSwitcher(t *testing.T) {
	videoPub := &recordingPublisher{}
	xmlPub := &recordingPublisher{}

	fc := NewStreamResponseSwitcher(func(_ context.Context, _ Promise, event Event) Publisher {
		switch event.Name() {
		case "video.parse":
			return videoPub
		case "xml.parse":
			return xmlPub
		default:
			return nil
		}
	}).(*streamResponseSwitcher)
	mux := NewTaskMux()
	fc.SetMux(mux)

	promise := mux.Handle("video.parse", testTask)
	wr := fc.Borrow(context.Background(), promise, WithPayload("video.parse", "value"))
	assert.NoError(t, wr.WriteResonse("hello"))
	assert.Len(t, videoPub.messages, 1)
	assert.Len(t, xmlPub.messages, 0)
	assert.NoError(t, wr.Release())

	promise = mux.Handle("xml.parse", testTask)
	wr = fc.Borrow(context.Background(), promise, WithPayload("xml.parse", "value"))
	assert.NoError(t, wr.WriteResonse("world"))
	assert.Len(t, videoPub.messages, 1)
	assert.Len(t, xmlPub.messages, 1)

	// Unmatched events are silently dropped when the selector returns nil.
	promise = mux.Handle("unknown", testTask)
	wr = fc.Borrow(context.Background(), promise, WithPayload("unknown", "value"))
	assert.NoError(t, wr.WriteResonse("dropped"))
	assert.Len(t, videoPub.messages, 1)
	assert.Len(t, xmlPub.messages, 1)

	assert.Panics(t, func() { NewStreamResponseSwitcher(nil) })
}
