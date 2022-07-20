package asyncp

import (
	"context"
	"log"
	"sync"

	"github.com/geniusrabbit/notificationcenter/v2"
)

// Receiver defines the processing interface.
// This handler used for processing of the input messages from the stream
type Receiver = notificationcenter.Receiver

// Subscriber defines the interface of subscribing for some
// event stream processing
type Subscriber = notificationcenter.Subscriber

// ProxySubscriber defineds the multiple subscriber object
type ProxySubscriber struct {
	subs []Subscriber
}

// NewProxySubscriber object from list of subscribers
func NewProxySubscriber(subs ...Subscriber) Subscriber {
	if len(subs) == 1 {
		return subs[0]
	}
	return &ProxySubscriber{subs: subs}
}

// Subscribe new handler for the stream
func (prx *ProxySubscriber) Subscribe(ctx context.Context, h Receiver) error {
	for _, sub := range prx.subs {
		if err := sub.Subscribe(ctx, h); err != nil {
			return err
		}
	}
	return nil
}

// Listen starts processing queue
func (prx *ProxySubscriber) Listen(ctx context.Context) (err error) {
	var w sync.WaitGroup
	for _, sub := range prx.subs {
		w.Add(1)
		go func(sub Subscriber) {
			if err = sub.Listen(ctx); err != nil {
				log.Printf("%t: %s", sub, err.Error())
			}
			w.Done()
		}(sub)
	}
	w.Wait()
	return err
}

// Close all proxy subscribers
func (prx *ProxySubscriber) Close() error {
	for _, sub := range prx.subs {
		_ = sub.Close()
	}
	return nil
}
