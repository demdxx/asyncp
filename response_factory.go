package asyncp

import "sync"

// ResponseWriterFactory interface to generate new response object
type ResponseWriterFactory interface {
	// Borrow response writer by event and task object
	Borrow(promise Promise, event Event) ResponseWriter

	// Release response writer object
	Release(w ResponseWriter)
}

type streamResponseFactory struct {
	pool      sync.Pool
	publisher Publisher
}

// NewStreamResponseFactory returns
func NewStreamResponseFactory(publisher Publisher) ResponseWriterFactory {
	return &streamResponseFactory{
		publisher: publisher,
		pool: sync.Pool{
			New: func() interface{} { return &responseStreamWriter{} },
		},
	}
}

func (s *streamResponseFactory) Borrow(promise Promise, event Event) ResponseWriter {
	wr := s.pool.Get().(*responseStreamWriter)
	wr.event = event
	wr.promise = promise
	wr.wstream = s.publisher
	return wr
}

func (s *streamResponseFactory) Release(w ResponseWriter) {
	if w == nil {
		return
	}
	wr := w.(*responseStreamWriter)
	wr.event = nil
	wr.wstream = nil
	s.pool.Put(wr)
}
