package asyncp

import (
	"context"
)

type responseWriterRelseasePool interface {
	// Release response writer object
	Release(w ResponseWriter)
}

// ResponseWriter basic response functionality
type ResponseWriter interface {
	// WriteResonse sends data into the stream response
	WriteResonse(response interface{}) error

	// RepeatWithResponse send data into the same stream response
	RepeatWithResponse(response interface{}) error

	// Release response writer stream
	Release() error
}

// ResponseHandlerFnk provides implementation of ResponseWriter interface
type ResponseHandlerFnk func(response interface{}) error

// WriteResonse sends data into the stream response
func (f ResponseHandlerFnk) WriteResonse(response interface{}) error {
	return f(response)
}

// RepeatWithResponse send data into the same stream response
func (f ResponseHandlerFnk) RepeatWithResponse(response interface{}) error {
	return f(response)
}

// Release response writer stream empty method
func (f ResponseHandlerFnk) Release() error {
	return nil
}

type responseProxyWriter struct {
	event   Event
	promise Promise
	mux     *TaskMux
	pool    responseWriterRelseasePool
}

func (wr *responseProxyWriter) WriteResonse(value interface{}) error {
	var (
		errs   multiError
		events = wr.promise.TargetEventName()
	)
	for _, eventName := range events {
		errs.Add(wr.writeResonseWithEventName(eventName, value, false))
	}
	if len(events) == 0 {
		errs.Add(wr.writeResonseWithEventName("", value, false))
	}
	return errs.AsError()
}

func (wr *responseProxyWriter) RepeatWithResponse(value interface{}) error {
	return wr.writeResonseWithEventName(wr.promise.EventName(), value, true)
}

func (wr *responseProxyWriter) writeResonseWithEventName(name string, value interface{}, repeat bool) error {
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
	return wr.mux.ExecuteEvent(ev)
}

func (wr *responseProxyWriter) Release() error {
	if wr.pool != nil {
		wr.pool.Release(wr)
	}
	return nil
}

type responseStreamWriter struct {
	ctx     context.Context
	event   Event
	promise Promise
	wstream Publisher
	pool    responseWriterRelseasePool
	mux     *TaskMux
}

func (wr *responseStreamWriter) WriteResonse(value interface{}) error {
	var (
		errs   multiError
		events = wr.promise.TargetEventName()
	)
	for _, eventName := range events {
		errs.Add(wr.writeResonseWithEventName(eventName, value, false))
	}
	if len(events) == 0 {
		errs.Add(wr.writeResonseWithEventName("", value, false))
	}
	return errs.AsError()
}

func (wr *responseStreamWriter) RepeatWithResponse(value interface{}) error {
	return wr.writeResonseWithEventName(wr.promise.EventName(), value, true)
}

func (wr *responseStreamWriter) writeResonseWithEventName(name string, value interface{}, repeat bool) error {
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
	return wr.wstream.Publish(wr.getExecContext(), ev)
}

func (wr *responseStreamWriter) Release() error {
	if wr.pool != nil {
		wr.pool.Release(wr)
	}
	return nil
}

func (wr *responseStreamWriter) getExecContext() context.Context {
	if wr.ctx != nil {
		return wr.ctx
	}
	return context.Background()
}
