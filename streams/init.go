package streams

import (
	"context"
	"fmt"
	"net/url"

	nc "github.com/geniusrabbit/notificationcenter/v2"
)

var subscribers = map[string]func(context.Context, string) (nc.Subscriber, error){}

// SubscriberFromURL establish new connection to the specific type of stream
func SubscriberFromURL(ctx context.Context, connURL string) (nc.Subscriber, error) {
	u, err := url.Parse(connURL)
	if err != nil {
		return nil, err
	}
	f := subscribers[u.Scheme]
	if f == nil {
		return nil, fmt.Errorf("unsupported subscriber %s", u.Scheme)
	}
	return f(ctx, connURL)
}
