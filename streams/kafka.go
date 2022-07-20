//go:build !apnekafka
// +build !apnekafka

package streams

import (
	"context"

	nc "github.com/geniusrabbit/notificationcenter/v2"
	"github.com/geniusrabbit/notificationcenter/v2/kafka"
)

func init() {
	subscribers["kafka"] = func(_ context.Context, conn string) (nc.Subscriber, error) {
		return kafka.NewSubscriber(kafka.WithKafkaURL(conn))
	}
}
