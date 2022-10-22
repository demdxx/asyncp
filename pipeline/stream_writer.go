package pipeline

import (
	"sync"

	"github.com/demdxx/asyncp/v2"
	"github.com/pkg/errors"
)

// ErrResponseRepeatUnsupported in case of pipeline
var ErrResponseRepeatUnsupported = errors.New("response repeat unsupported in pipeline")

type eventPoolItem struct {
	data []asyncp.Event
}

var eventPool = &sync.Pool{
	New: func() any {
		return &eventPoolItem{data: make([]asyncp.Event, 0, 100)}
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
		pool:        eventPool.Get().(*eventPoolItem).data,
	}
}

// WriteEvent sends event into the stream response
func (stream *stream) WriteResonse(response any) error {
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

// RepeatWithResponse send data into the same stream response
func (stream *stream) RepeatWithResponse(response any) error {
	return ErrResponseRepeatUnsupported
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
	eventPool.Put(&eventPoolItem{data: stream.pool})
	return nil
}

func (stream *stream) Release() error {
	return nil
}
