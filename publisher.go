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
