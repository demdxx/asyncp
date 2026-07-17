package asyncp

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestEventMethods(t *testing.T) {
	event := WithPayload(`test`, 100)

	errEvent := event.WithError(fmt.Errorf(`test`))
	assert.NotNil(t, errEvent.Err(), `error`)

	event.SetComplete(true)
	assert.True(t, event.IsComplete())

	event2 := event.WithName(`test2`)
	assert.Equal(t, `test2`, event2.Name())

	assert.NotEqual(t, ``, event2.String())
}

func TestEventAfter(t *testing.T) {
	event1 := WithPayload(`test1`, 100)
	event2 := WithPayload(`test2`, 100).After(event1)
	event3 := WithPayload(`test3`, 100).After(event2)

	assert.ElementsMatch(t, []string{`test1`, `test2`}, event3.DoneTasks())
}

func TestEventRepeat(t *testing.T) {
	event1 := WithPayload(`test1`, 100).After(WithPayload(`test0`, nil))
	sent1, repeated1 := event1.Counters()

	event2 := WithPayload(`test2`, 200).Repeat(event1)
	sent2, repeated2 := event2.Counters()

	assert.Equal(t, sent1+1, sent2)
	assert.Equal(t, repeated1+1, repeated2)
	assert.ElementsMatch(t, event1.DoneTasks(), event2.DoneTasks())
}

func TestEventAccessors(t *testing.T) {
	mux := NewTaskMux()
	promise := mux.Handle(`test`, testTask)

	event := WithPayload(`test`, 100)
	assert.NotEqual(t, uuid.Nil, event.ID())
	assert.WithinDuration(t, time.Now(), event.CreatedAt(), time.Second)

	assert.Nil(t, event.Mux())
	event.SetMux(mux)
	assert.Same(t, mux, event.Mux())

	assert.Nil(t, event.Promise())
	event.SetPromise(promise)
	assert.Same(t, promise, event.Promise())

	assert.False(t, event.HasDoneTask(`nope`))
}

func TestEventEncodeDecodeRoundTrip(t *testing.T) {
	original := WithPayload(`test`, map[string]any{`a`: float64(1)}).(*event)
	original.SetComplete(true)

	data, err := original.MarshalJSON()
	assert.NoError(t, err)

	decoded := &event{}
	assert.NoError(t, decoded.UnmarshalJSON(data))
	assert.Equal(t, original.ID(), decoded.ID())
	assert.Equal(t, original.Name(), decoded.Name())
}

func TestEventDecodeInvalidJSON(t *testing.T) {
	ev := &event{}
	err := ev.Decode([]byte(`not-json`))
	assert.Error(t, err)
}

func TestEventClearResetsAllFields(t *testing.T) {
	mux := NewTaskMux()
	promise := mux.Handle(`test`, testTask)

	ev := WithPayload(`test`, 100).(*event)
	ev.SetMux(mux)
	ev.SetPromise(promise)
	ev.SetComplete(true)
	ev = ev.After(WithPayload(`prev`, nil)).(*event)

	ev.Clear()

	assert.Equal(t, uuid.Nil, ev.id)
	assert.Equal(t, ``, ev.name)
	assert.Nil(t, ev.mux)
	assert.Nil(t, ev.promise)
	assert.Nil(t, ev.payload)
	assert.Empty(t, ev.doneEvents)
	assert.Zero(t, ev.sendCount)
	assert.Zero(t, ev.retranslateCount)
	assert.False(t, ev.complete)
	assert.True(t, ev.createdAt.IsZero())
}
