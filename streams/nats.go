//go:build !apnenats

package streams

import (
	"context"

	nc "github.com/geniusrabbit/notificationcenter"
	"github.com/geniusrabbit/notificationcenter/nats"
)

func init() {
	subscribers["nats"] = func(_ context.Context, conn string) (nc.Subscriber, error) {
		return nats.NewSubscriber(nats.WithNatsURL(conn))
	}
}
