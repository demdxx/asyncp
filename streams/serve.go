package streams

import (
	"context"
	"log"

	"github.com/demdxx/asyncp"
	nc "github.com/geniusrabbit/notificationcenter"
)

// ListenAndServe task service for sources
func ListenAndServe(ctx context.Context, srv *asyncp.TaskMux, sources ...interface{}) error {
	subscribers := make([]nc.Subscriber, 0, len(sources))
	for _, src := range sources {
		switch v := src.(type) {
		case string:
			nsub, err := SubscriberFromURL(ctx, v)
			if err != nil {
				return err
			}
			subscribers = append(subscribers, nsub)
		case nc.Subscriber:
			subscribers = append(subscribers, v)
		}
	}
	subs := asyncp.NewProxySubscriber(subscribers...)
	defer func() {
		if err := subs.Close(); err != nil {
			log.Print(err.Error())
		}
	}()
	if err := subs.Subscribe(ctx, srv); err != nil {
		return err
	}
	return subs.Listen(ctx)
}
