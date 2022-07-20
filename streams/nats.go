//go:build !apnenats
// +build !apnenats

package streams

import (
	"context"

	nc "github.com/geniusrabbit/notificationcenter/v2"
	"github.com/geniusrabbit/notificationcenter/v2/nats"
)

func init() {
	subscribers["nats"] = func(_ context.Context, conn string) (nc.Subscriber, error) {
		return nats.NewSubscriber(nats.WithNatsURL(conn))
	}
}
