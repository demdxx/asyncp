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

// newResponseWriterPool returns a pool of the shared responseWriter type
// used by every factory implementation in this file.
func newResponseWriterPool() sync.Pool {
	return sync.Pool{New: func() any { return &responseWriter{} }}
}

// borrowFromPool takes a writer out of the pool and fills in the common fields.
func borrowFromPool(pool *sync.Pool, ctx context.Context, mux *TaskMux, releasePool responseWriterRelseasePool, promise Promise, event Event) *responseWriter {
	wr := pool.Get().(*responseWriter)
	wr.ctx = ctx
	wr.event = event
	wr.promise = promise
	wr.mux = mux
	wr.pool = releasePool
	return wr
}

// releaseToPool clears and returns the writer to the pool, mirroring the
// previous behaviour of only pooling writers that were actually borrowed.
func releaseToPool(pool *sync.Pool, w ResponseWriter) {
	if w == nil {
		return
	}
	if wr, ok := w.(*responseWriter); ok && wr.event != nil {
		wr.reset()
		pool.Put(wr)
	}
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
		pool:      newResponseWriterPool(),
	}
}

func (s *streamResponseFactory) SetMux(mux *TaskMux) {
	s.mux = mux
}

func (s *streamResponseFactory) Borrow(ctx context.Context, promise Promise, event Event) ResponseWriter {
	wr := borrowFromPool(&s.pool, ctx, s.mux, s, promise, event)
	wr.dispatch = func(ctx context.Context, ev Event) error {
		return s.publisher.Publish(ctx, ev)
	}
	return wr
}

func (s *streamResponseFactory) Release(w ResponseWriter) {
	releaseToPool(&s.pool, w)
}

type proxyResponseFactory struct {
	mux  *TaskMux
	pool sync.Pool
}

// NewProxyResponseFactory returns
func NewProxyResponseFactory() ResponseWriterFactory {
	return &proxyResponseFactory{
		pool: newResponseWriterPool(),
	}
}

func (s *proxyResponseFactory) SetMux(mux *TaskMux) {
	s.mux = mux
}

func (s *proxyResponseFactory) Borrow(ctx context.Context, promise Promise, event Event) ResponseWriter {
	wr := borrowFromPool(&s.pool, ctx, s.mux, s, promise, event)
	wr.dispatch = func(_ context.Context, ev Event) error {
		return wr.mux.ExecuteEvent(ev)
	}
	return wr
}

func (s *proxyResponseFactory) Release(w ResponseWriter) {
	releaseToPool(&s.pool, w)
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
		pool:       newResponseWriterPool(),
	}
}

func (s *mutistreamResponseFactory) SetMux(mux *TaskMux) {
	s.mux = mux
}

func (s *mutistreamResponseFactory) Borrow(ctx context.Context, promise Promise, event Event) ResponseWriter {
	wr := borrowFromPool(&s.pool, ctx, s.mux, s, promise, event)
	pub := s.publisher(event)
	wr.dispatch = func(ctx context.Context, ev Event) error {
		if pub == nil {
			return nil
		}
		return pub.Publish(ctx, ev)
	}
	return wr
}

func (s *mutistreamResponseFactory) Release(w ResponseWriter) {
	releaseToPool(&s.pool, w)
}

func (s *mutistreamResponseFactory) publisher(event Event) Publisher {
	for _, item := range s.publishers {
		if item.test(event.Name()) {
			return item.publisher
		}
	}
	return nil
}

// StreamPublisherSelectorFnk picks the Publisher that should receive the
// response of the given promise/event pair. It's evaluated once per borrowed
// response writer, so the decision may depend on the event name, its
// payload, or the promise it belongs to.
type StreamPublisherSelectorFnk func(ctx context.Context, promise Promise, event Event) Publisher

type streamResponseSwitcher struct {
	pool     sync.Pool
	selector StreamPublisherSelectorFnk
	mux      *TaskMux
}

// NewStreamResponseSwitcher returns a ResponseWriterFactory which routes each
// response to a Publisher picked at runtime by the selector function. Unlike
// NewMultistreamResponseFactory (pattern/regexp matching on the event name),
// the selector receives full context and can implement arbitrary branching
// logic, e.g. sending `video.parse` events to one stream and `xml.parse`
// events to another:
//
//	asyncp.NewStreamResponseSwitcher(func(_ context.Context, _ asyncp.Promise, event asyncp.Event) asyncp.Publisher {
//		switch event.Name() {
//		case "video.parse":
//			return videoPublisher
//		case "xml.parse":
//			return xmlPublisher
//		default:
//			return defaultPublisher
//		}
//	})
//
// If the selector returns nil, the response is silently dropped.
func NewStreamResponseSwitcher(selector StreamPublisherSelectorFnk) ResponseWriterFactory {
	if selector == nil {
		panic("asyncp: NewStreamResponseSwitcher requires a non-nil selector")
	}
	return &streamResponseSwitcher{
		selector: selector,
		pool:     newResponseWriterPool(),
	}
}

func (s *streamResponseSwitcher) SetMux(mux *TaskMux) {
	s.mux = mux
}

func (s *streamResponseSwitcher) Borrow(ctx context.Context, promise Promise, event Event) ResponseWriter {
	wr := borrowFromPool(&s.pool, ctx, s.mux, s, promise, event)
	pub := s.selector(ctx, promise, event)
	wr.dispatch = func(ctx context.Context, ev Event) error {
		if pub == nil {
			return nil
		}
		return pub.Publish(ctx, ev)
	}
	return wr
}

func (s *streamResponseSwitcher) Release(w ResponseWriter) {
	releaseToPool(&s.pool, w)
}
