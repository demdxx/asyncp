package streams

import (
	"context"
	"runtime"

	nc "github.com/geniusrabbit/notificationcenter/v2"
	"github.com/geniusrabbit/notificationcenter/v2/gochan"
)

func init() {
	proxy := gochan.New(max(runtime.NumCPU(), 1))
	subscribers["gochan"] = func(ctx context.Context, url string) (nc.Subscriber, error) { return proxy, nil }
	subscribers["chan"] = subscribers["gochan"]
}
