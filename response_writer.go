package asyncp

import "context"

// ResponseWriter basic response functionality
type ResponseWriter interface {
	// WriteResonse sends data into the stream response
	WriteResonse(response interface{}) error
}

// ResponseHandlerFnk provides implementation of ResponseWriter interface
type ResponseHandlerFnk func(response interface{}) error

// WriteResonse sends data into the stream response
func (f ResponseHandlerFnk) WriteResonse(response interface{}) error {
	return f(response)
}

type responseProxyWriter struct {
	parent  Event
	promise Promise
	mux     *TaskMux
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

type responseStreamWriter struct {
	event   Event
	promise Promise
	wstream Publisher
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
	return wr.wstream.Publish(context.Background(), ev)
}
