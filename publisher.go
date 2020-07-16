package asyncp

import (
	"context"

	"github.com/geniusrabbit/notificationcenter"
)

// Publisher writing interface
type Publisher = notificationcenter.Publisher

// Retranslator of the event to the stream
func Retranslator(pub Publisher) Task {
	return FuncTask(func(ctx context.Context, event Event, responseWriter ResponseWriter) error {
		return pub.Publish(ctx, event)
	})
}

type publisherEventWrapper struct {
	name string
	pub  Publisher
}

// PublisherEventWrapper with fixed event name
func PublisherEventWrapper(eventName string, publisher Publisher) Publisher {
	return &publisherEventWrapper{
		name: eventName,
		pub:  publisher,
	}
}

func (wr *publisherEventWrapper) Publish(ctx context.Context, messages ...interface{}) error {
	events := make([]interface{}, 0, len(messages))
	for _, msg := range messages {
		events = append(events, WithPayload(wr.name, msg))
	}
	return wr.pub.Publish(ctx, events...)
}
