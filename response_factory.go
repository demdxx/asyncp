package asyncp

import (
	"context"
	"fmt"
	"regexp"
	"sync"
)

// ResponseWriterFactory interface to generate new response object
type ResponseWriterFactory interface {
	// Borrow response writer by event and task object
	Borrow(ctx context.Context, promise Promise, event Event) ResponseWriter

	// Release response writer object
	Release(w ResponseWriter)
}

type streamResponseFactory struct {
	pool      sync.Pool
	publisher Publisher
	mux       *TaskMux
}

// NewStreamResponseFactory returns
func NewStreamResponseFactory(publisher Publisher) ResponseWriterFactory {
	return &streamResponseFactory{
		publisher: publisher,
		pool: sync.Pool{
			New: func() any { return &responseStreamWriter{} },
		},
	}
}

func (s *streamResponseFactory) SetMux(mux *TaskMux) {
	s.mux = mux
}

func (s *streamResponseFactory) Borrow(ctx context.Context, promise Promise, event Event) ResponseWriter {
	wr := s.pool.Get().(*responseStreamWriter)
	wr.ctx = ctx
	wr.event = event
	wr.promise = promise
	wr.wstream = s.publisher
	wr.mux = s.mux
	wr.pool = s
	return wr
}

func (s *streamResponseFactory) Release(w ResponseWriter) {
	if w == nil {
		return
	}
	if wr := w.(*responseStreamWriter); wr.event != nil {
		wr.ctx = nil
		wr.event = nil
		wr.wstream = nil
		wr.pool = nil
		wr.mux = nil
		s.pool.Put(wr)
	}
}

type proxyResponseFactory struct {
	mux  *TaskMux
	pool sync.Pool
}

// NewProxyResponseFactory returns
func NewProxyResponseFactory() ResponseWriterFactory {
	return &proxyResponseFactory{
		pool: sync.Pool{
			New: func() any { return &responseProxyWriter{} },
		},
	}
}

func (s *proxyResponseFactory) SetMux(mux *TaskMux) {
	s.mux = mux
}

func (s *proxyResponseFactory) Borrow(ctx context.Context, promise Promise, event Event) ResponseWriter {
	wr := s.pool.Get().(*responseProxyWriter)
	wr.event = event
	wr.promise = promise
	wr.pool = s
	wr.mux = s.mux
	return wr
}

func (s *proxyResponseFactory) Release(w ResponseWriter) {
	if w == nil {
		return
	}
	if wr := w.(*responseProxyWriter); wr.event != nil {
		wr.promise = nil
		wr.event = nil
		wr.pool = nil
		wr.mux = nil
		s.pool.Put(wr)
	}
}

type multistreamItem struct {
	tfn       []func(string) bool
	patterns  []*regexp.Regexp
	publisher Publisher
}

func (it *multistreamItem) test(eventName string) bool {
	if len(it.patterns) == 0 && len(it.tfn) == 0 {
		return true
	}
	for _, f := range it.tfn {
		if f(eventName) {
			return true
		}
	}
	for _, ptr := range it.patterns {
		if ptr.MatchString(eventName) {
			return true
		}
	}
	return false
}

type mutistreamResponseFactory struct {
	pool       sync.Pool
	publishers []*multistreamItem
	mux        *TaskMux
}

// NewMultistreamResponseFactory returns implementation with multipublisher support
func NewMultistreamResponseFactory(streams ...any) ResponseWriterFactory {
	var (
		publishers []*multistreamItem
		commonItem *multistreamItem
		item       = new(multistreamItem)
	)
	for _, v := range streams {
		switch val := v.(type) {
		case []func(string) bool:
			item.tfn = append(item.tfn, val...)
		case func(string) bool:
			item.tfn = append(item.tfn, val)
		case []string:
			for _, ptr := range val {
				item.patterns = append(item.patterns, regexp.MustCompile(ptr))
			}
		case string:
			item.patterns = append(item.patterns, regexp.MustCompile(val))
		case *regexp.Regexp:
			item.patterns = append(item.patterns, val)
		case []*regexp.Regexp:
			item.patterns = append(item.patterns, val...)
		case Publisher:
			item.publisher = val
			if len(item.patterns) == 0 {
				if commonItem != nil {
					panic("only one default publisher supports")
				}
				commonItem = item
			} else {
				publishers = append(publishers, item)
			}
			item = new(multistreamItem)
		default:
			panic(fmt.Sprintf("invalid parameter %T", v))
		}
	}
	if commonItem != nil {
		publishers = append(publishers, commonItem)
	}
	return &mutistreamResponseFactory{
		publishers: publishers,
		pool: sync.Pool{
			New: func() any { return &responseStreamWriter{} },
		},
	}
}

func (s *mutistreamResponseFactory) SetMux(mux *TaskMux) {
	s.mux = mux
}

func (s *mutistreamResponseFactory) Borrow(ctx context.Context, promise Promise, event Event) ResponseWriter {
	wr := s.pool.Get().(*responseStreamWriter)
	wr.event = event
	wr.promise = promise
	wr.wstream = s.publisher(event)
	wr.mux = s.mux
	wr.pool = s
	return wr
}

func (s *mutistreamResponseFactory) Release(w ResponseWriter) {
	if w == nil {
		return
	}
	if wr := w.(*responseStreamWriter); wr.event != nil {
		wr.event = nil
		wr.wstream = nil
		wr.pool = nil
		wr.mux = nil
		s.pool.Put(wr)
	}
}

func (s *mutistreamResponseFactory) publisher(event Event) Publisher {
	for _, item := range s.publishers {
		if item.test(event.Name()) {
			return item.publisher
		}
	}
	return nil
}
