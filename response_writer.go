package asyncp

import (
	"context"

	"go.uber.org/multierr"
)

type responseWriterRelseasePool interface {
	// Release response writer object
	Release(w ResponseWriter)
}

// ResponseWriter basic response functionality
type ResponseWriter interface {
	// WriteResonse sends data into the stream response
	WriteResonse(response any) error

	// RepeatWithResponse send data into the same stream response
	RepeatWithResponse(response any) error

	// Release response writer stream
	Release() error
}

// ResponseHandlerFnk provides implementation of ResponseWriter interface
type ResponseHandlerFnk func(response any) error

// WriteResonse sends data into the stream response
func (f ResponseHandlerFnk) WriteResonse(response any) error {
	return f(response)
}

// RepeatWithResponse send data into the same stream response
func (f ResponseHandlerFnk) RepeatWithResponse(response any) error {
	return f(response)
}

// Release response writer stream empty method
func (f ResponseHandlerFnk) Release() error {
	return nil
}

// dispatchFnk delivers the prepared event to its final destination.
// It's the only thing that actually differs between the response writer
// flavours (proxy back into the local mux vs publish to an external
// stream), so every factory below shares the same `responseWriter` type
// and just plugs in a different dispatchFnk.
type dispatchFnk func(ctx context.Context, event Event) error

// responseWriter is the single ResponseWriter implementation shared by
// streamResponseFactory, proxyResponseFactory and mutistreamResponseFactory.
type responseWriter struct {
	ctx      context.Context
	event    Event
	promise  Promise
	mux      *TaskMux
	pool     responseWriterRelseasePool
	dispatch dispatchFnk
}

func (wr *responseWriter) WriteResonse(value any) error {
	var (
		err    error
		events = wr.promise.TargetEventName()
	)
	for _, eventName := range events {
		err = multierr.Append(err, wr.writeResonseWithEventName(eventName, value, false))
	}
	if len(events) == 0 {
		err = multierr.Append(err, wr.writeResonseWithEventName("", value, false))
	}
	return err
}

func (wr *responseWriter) RepeatWithResponse(value any) error {
	return wr.writeResonseWithEventName(wr.promise.EventName(), value, true)
}

func (wr *responseWriter) writeResonseWithEventName(name string, value any, repeat bool) error {
	var ev Event
	switch v := value.(type) {
	case Event:
		ev = v
	default:
		ev = wr.event.WithPayload(value)
	}
	if name != "" || !repeat {
		ev = ev.WithName(name)
	}
	if repeat {
		ev = ev.Repeat(wr.event)
	} else {
		ev = ev.After(wr.event)
	}
	ev.SetMux(wr.mux)
	return wr.dispatch(wr.execContext(), ev)
}

func (wr *responseWriter) Release() error {
	if wr.pool != nil {
		wr.pool.Release(wr)
	}
	return nil
}

func (wr *responseWriter) execContext() context.Context {
	if wr.ctx != nil {
		return wr.ctx
	}
	return context.Background()
}

// reset clears all borrowed state before the writer goes back to its pool.
func (wr *responseWriter) reset() {
	wr.ctx = nil
	wr.event = nil
	wr.promise = nil
	wr.mux = nil
	wr.pool = nil
	wr.dispatch = nil
}
