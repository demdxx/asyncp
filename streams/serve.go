package streams

import (
	"context"
	"fmt"
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
	fmt.Println("GGG-0", subs != nil)
	if err := subs.Subscribe(ctx, srv); err != nil {
		return err
	}
	fmt.Println("GGG-1")
	e := subs.Listen(ctx)
	fmt.Println("GGG-2", e)
	return e
}
