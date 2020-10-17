package asyncp

import "context"

type responseWriterRelseasePool interface {
	// Release response writer object
	Release(w ResponseWriter)
}

// ResponseWriter basic response functionality
type ResponseWriter interface {
	// WriteResonse sends data into the stream response
	WriteResonse(response interface{}) error

	// Release response writer stream
	Release() error
}

// ResponseHandlerFnk provides implementation of ResponseWriter interface
type ResponseHandlerFnk func(response interface{}) error

// WriteResonse sends data into the stream response
func (f ResponseHandlerFnk) WriteResonse(response interface{}) error {
	return f(response)
}

// Release response writer stream empty method
func (f ResponseHandlerFnk) Release() error {
	return nil
}

type responseProxyWriter struct {
	parent  Event
	promise Promise
	mux     *TaskMux
	pool    responseWriterRelseasePool
}

func (wr *responseProxyWriter) WriteResonse(value interface{}) error {
	var ev Event
	switch v := value.(type) {
	case Event:
		ev = v
	default:
		ev = wr.parent.WithPayload(value)
	}
	if ev.IsComplete() {
		ev = ev.WithName(wr.promise.TargetEventName())
	}
	msg, err := messageFrom(ev)
	if err != nil {
		return err
	}
	return wr.mux.Receive(msg)
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
}

func (wr *responseStreamWriter) WriteResonse(value interface{}) error {
	var ev Event
	switch v := value.(type) {
	case Event:
		ev = v
	default:
		ev = wr.event.WithPayload(value)
	}
	if ev.IsComplete() {
		ev = ev.WithName(wr.promise.TargetEventName())
	}
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
