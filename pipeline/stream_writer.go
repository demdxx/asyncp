package pipeline

import (
	"sync"

	"github.com/demdxx/asyncp"
)

var eventPool = &sync.Pool{
	New: func() interface{} {
		return make([]asyncp.Event, 0, 100)
	},
}

// StreamWriter implements basic Writer interface
type stream struct {
	mx sync.RWMutex

	cursor      int
	parentEvent asyncp.Event
	pool        []asyncp.Event
}

func newStream(event asyncp.Event) *stream {
	return &stream{
		cursor:      0,
		parentEvent: event,
		pool:        eventPool.Get().([]asyncp.Event),
	}
}

// WriteEvent sends event into the stream response
func (stream *stream) WriteResonse(response interface{}) error {
	stream.mx.Lock()
	defer stream.mx.Unlock()
	switch r := response.(type) {
	case asyncp.Event:
		stream.pool = append(stream.pool, r)
	default:
		stream.pool = append(stream.pool, stream.parentEvent.WithPayload(response))
	}
	return nil
}

func (stream *stream) nextEvent() asyncp.Event {
	if stream.cursor >= len(stream.pool) {
		return nil
	}
	stream.cursor++
	return stream.pool[stream.cursor-1]
}

func (stream *stream) Reset() {
	stream.pool = stream.pool[:0]
	stream.cursor = 0
}

func (stream *stream) Close() error {
	stream.Reset()
	//lint:ignore SA6002 Using pointer would allocate more since we would have to copy slice header before taking a pointer.
	eventPool.Put(stream.pool)
	return nil
}
