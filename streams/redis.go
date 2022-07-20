//go:build !apneredis
// +build !apneredis

package streams

import (
	"context"

	nc "github.com/geniusrabbit/notificationcenter/v2"
	"github.com/geniusrabbit/notificationcenter/v2/redis"
)

func init() {
	subscribers["redis"] = func(_ context.Context, connURL string) (nc.Subscriber, error) {
		return redis.NewSubscriber(redis.WithRedisURL(connURL))
	}
}
