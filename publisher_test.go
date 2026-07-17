package asyncp

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type recordingPublisher struct {
	messages []any
	err      error
}

func (p *recordingPublisher) Publish(ctx context.Context, messages ...any) error {
	if p.err != nil {
		return p.err
	}
	p.messages = append(p.messages, messages...)
	return nil
}

func newResponseRecorder() (*[]any, ResponseWriter) {
	var received []any
	return &received, ResponseHandlerFnk(func(response any) error {
		received = append(received, response)
		return nil
	})
}

func TestRetranslator(t *testing.T) {
	pub := &recordingPublisher{}
	received, rw := newResponseRecorder()
	task := Retranslator(0, pub)

	event := WithPayload("test", "payload")
	err := task.Execute(context.Background(), event, rw)
	assert.NoError(t, err)
	assert.Len(t, pub.messages, 1)
	if assert.Len(t, *received, 1) {
		assert.Same(t, event, (*received)[0])
	}
}

func TestRetranslatorPublishError(t *testing.T) {
	pub := &recordingPublisher{err: errors.New("publish failed")}
	_, rw := newResponseRecorder()
	task := Retranslator(DefaultRetranslateCount, pub)

	err := task.Execute(context.Background(), WithPayload("test", "payload"), rw)
	assert.ErrorContains(t, err, "publish failed")
}

func TestRetranslatorSkipAfterMaxRepeats(t *testing.T) {
	pub := &recordingPublisher{}
	received, rw := newResponseRecorder()
	task := Retranslator(1, pub)

	event := WithPayload("test", "payload")
	// Simulate an event which has already been retranslated too many times.
	event = event.Repeat(event)
	event = event.Repeat(event)

	err := task.Execute(context.Background(), event, rw)
	assert.NoError(t, err)
	assert.Empty(t, pub.messages, "should not publish once repeat limit exceeded")
	if assert.Len(t, *received, 1) {
		assert.ErrorIs(t, (*received)[0].(Event).Err(), ErrSkipEvent)
	}
}

func TestRepeaterEmptyEventName(t *testing.T) {
	_, rw := newResponseRecorder()
	task := Repeater()
	err := task.Execute(context.Background(), WithPayload("", "payload"), rw)
	assert.NoError(t, err)
}

func TestRepeaterSkipAfterMaxRepeats(t *testing.T) {
	received, rw := newResponseRecorder()
	task := Repeater(1)

	event := WithPayload("test", "payload")
	event = event.Repeat(event)
	event = event.Repeat(event)

	err := task.Execute(context.Background(), event, rw)
	assert.NoError(t, err)
	if assert.Len(t, *received, 1) {
		assert.ErrorIs(t, (*received)[0].(Event).Err(), ErrSkipEvent)
	}
}

func TestRepeaterRepeatsEvent(t *testing.T) {
	mux := NewTaskMux()
	var repeated Event
	mux.Handle("test", FuncTask(func(_ context.Context, e Event, rw ResponseWriter) error {
		repeated = e
		return nil
	}))

	wr := &responseWriter{mux: mux, event: WithPayload("test", "payload"), promise: mux.tasks["test"]}
	wr.dispatch = func(_ context.Context, ev Event) error { return mux.ExecuteEvent(ev) }

	task := Repeater(DefaultRetranslateCount)
	err := task.Execute(context.Background(), WithPayload("test", "payload"), wr)
	assert.NoError(t, err)
	assert.NotNil(t, repeated)
}

func TestPublisherEventWrapper(t *testing.T) {
	pub := &recordingPublisher{}
	wr := PublisherEventWrapper("wrapped", pub).(*publisherEventWrapper)
	mux := NewTaskMux()
	wr.SetMux(mux)

	err := wr.Publish(context.Background(), "hello", "world")
	assert.NoError(t, err)
	if assert.Len(t, pub.messages, 2) {
		for _, m := range pub.messages {
			ev := m.(Event)
			assert.Equal(t, "wrapped", ev.Name())
		}
	}
}

func TestPublisherEventWrapperReturnIDs(t *testing.T) {
	pub := &recordingPublisher{}
	wr := PublisherEventWrapper("wrapped", pub)

	ids, err := wr.PublishAndReturnIDs(context.Background(), "hello", "world")
	assert.NoError(t, err)
	assert.Len(t, ids, 2)
	assert.Len(t, pub.messages, 2)
}

func TestPublisherEventWrapperReturnIDsError(t *testing.T) {
	pub := &recordingPublisher{err: errors.New("boom")}
	wr := PublisherEventWrapper("wrapped", pub)

	ids, err := wr.PublishAndReturnIDs(context.Background(), "hello")
	assert.Error(t, err)
	assert.Nil(t, ids)
}
