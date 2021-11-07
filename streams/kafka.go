//go:build !apnekafka
// +build !apnekafka

package streams

import (
	"context"

	nc "github.com/geniusrabbit/notificationcenter"
	"github.com/geniusrabbit/notificationcenter/kafka"
)

func init() {
	subscribers["kafka"] = func(_ context.Context, conn string) (nc.Subscriber, error) {
		return kafka.NewSubscriber(kafka.WithKafkaURL(conn))
	}
}
