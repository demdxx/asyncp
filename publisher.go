package asyncp

import (
	"context"

	"github.com/geniusrabbit/notificationcenter"
)

// Publisher writing interface
type Publisher = notificationcenter.Publisher

// Retranslator of the event to the stream
func Retranslator(pubs ...Publisher) Task {
	return FuncTask(func(ctx context.Context, event Event, responseWriter ResponseWriter) error {
		for _, pub := range pubs {
			if err := pub.Publish(ctx, event); err != nil {
				return err
			}
		}
		return responseWriter.WriteResonse(event)
	})
}

type publisherEventWrapper struct {
	name string
	pub  Publisher
	mux  *TaskMux
}

// PublisherEventWrapper with fixed event name
func PublisherEventWrapper(eventName string, publisher Publisher) Publisher {
	return &publisherEventWrapper{
		name: eventName,
		pub:  publisher,
	}
}

func (wr *publisherEventWrapper) SetMux(mux *TaskMux) {
	wr.mux = mux
}

func (wr *publisherEventWrapper) Publish(ctx context.Context, messages ...interface{}) error {
	events := make([]interface{}, 0, len(messages))
	for _, msg := range messages {
		event := WithPayload(wr.name, msg)
		event.SetMux(wr.mux)
		events = append(events, event)
	}
	return wr.pub.Publish(ctx, events...)
}
