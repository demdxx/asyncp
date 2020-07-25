package asyncp

import (
	"fmt"
	"regexp"
	"sync"
)

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
	wr.pool = s
	return wr
}

func (s *streamResponseFactory) Release(w ResponseWriter) {
	if w == nil {
		return
	}
	if wr := w.(*responseStreamWriter); wr.event != nil {
		wr.event = nil
		wr.wstream = nil
		wr.pool = nil
		s.pool.Put(wr)
	}
}

type multistreamItem struct {
	patterns  []*regexp.Regexp
	publisher Publisher
}

func (it *multistreamItem) test(eventName string) bool {
	if len(it.patterns) == 0 {
		return true
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
}

// NewMultistreamResponseFactory returns implementation with multipublisher support
func NewMultistreamResponseFactory(streams ...interface{}) ResponseWriterFactory {
	var (
		publishers []*multistreamItem
		commonItem *multistreamItem
		item       = new(multistreamItem)
	)
	for _, v := range streams {
		switch val := v.(type) {
		case string:
			item.patterns = append(item.patterns, regexp.MustCompile(val))
		case *regexp.Regexp:
			item.patterns = append(item.patterns, val)
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
			New: func() interface{} { return &responseStreamWriter{} },
		},
	}
}

func (s *mutistreamResponseFactory) Borrow(promise Promise, event Event) ResponseWriter {
	wr := s.pool.Get().(*responseStreamWriter)
	wr.event = event
	wr.promise = promise
	wr.wstream = s.publisher(event)
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
