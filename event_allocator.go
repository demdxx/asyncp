package asyncp

import "sync"

// EventAllocator provides event allocation objects interface
type EventAllocator interface {
	Decode(msg Message) (Event, error)
	Release(event Event) error
}

type defaultEventAllocator struct {
	pool sync.Pool
}

func newDefaultEventAllocator() *defaultEventAllocator {
	return &defaultEventAllocator{
		pool: sync.Pool{New: func() any {
			return new(event)
		}},
	}
}

func (a *defaultEventAllocator) Decode(msg Message) (Event, error) {
	event := &event{}
	return event, event.Decode(msg.Body())
}

func (a *defaultEventAllocator) Release(e Event) error {
	if e != nil {
		ev := e.(*event)
		ev.Clear()
		a.pool.Put(ev)
	}
	return nil
}
