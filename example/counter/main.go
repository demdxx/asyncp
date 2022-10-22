package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"time"

	nc "github.com/geniusrabbit/notificationcenter/v2"
	"github.com/geniusrabbit/notificationcenter/v2/interval"

	"github.com/demdxx/asyncp/v2"
	"github.com/demdxx/asyncp/v2/monitor/driver/redis"
	"github.com/demdxx/asyncp/v2/monitor/kvstorage"
	"github.com/demdxx/asyncp/v2/streams"
)

var (
	storageFlag = flag.String("storage", "", "redis://host:port/dbnum")
	appnameFlag = flag.String("app", "test", "application name")
)

func main() {
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fmt.Println("> connect storage")
	redisDriver, err := redis.New(*storageFlag)
	if err != nil {
		log.Fatal(err)
	}
	redisStorage, err := kvstorage.New(
		kvstorage.WithKVClient(redisDriver),
		kvstorage.WithTaskDetailInfo(time.Second*60))
	if err != nil {
		log.Fatal(err)
	}

	iterator := 0
	mux := asyncp.NewTaskMux(
		asyncp.WithCluster(*appnameFlag,
			asyncp.ClusterWithReader(kvstorage.NewClusterInfoReader(redisDriver, *appnameFlag)),
			asyncp.ClusterWithStores(redisStorage)),
	)
	mux.Handle("count",
		func(_ context.Context, ev asyncp.Event, w asyncp.ResponseWriter) error {
			iterator++
			fmt.Println("TaskID:", ev.ID().String())
			fmt.Println("TaskName:", ev.Name())
			fmt.Println("Iter:", iterator)
			if rand.Float32() > 0.5 {
				return fmt.Errorf("random error for %d", iterator)
			}
			return w.WriteResonse(iterator)
		}).
		Then(func(_ context.Context, ev asyncp.Event, w asyncp.ResponseWriter) error {
			fmt.Println("TaskID:", ev.ID().String())
			fmt.Println("TaskName:", ev.Name())
			fmt.Println("Subtask:", iterator)
			// return mux.ExecuteEvent(ev.WithName("next-count").WithPayload(iterator))
			return w.WriteResonse(iterator)
		})
	// Execute task with name "next-count" right after the "count" task.
	// This king of events can't be used for parent mapping
	mux.Handle("count>next-count",
		func(_ context.Context, ev asyncp.Event, w asyncp.ResponseWriter) error {
			fmt.Println("TaskID:", ev.ID().String())
			fmt.Println("TaskName:", ev.Name())
			fmt.Println("Next:", iterator)
			return nil
		})
	if err := mux.FinishInit(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("> subscribe counter")
	sub := interval.NewSubscriber(time.Second,
		interval.WithHandler(func() any {
			return asyncp.WithPayload("count", iterator)
		}),
		interval.WithErrorHandler(func(_ nc.Message, err error) {}))

	fmt.Println("> run listener", sub != nil)
	_ = streams.ListenAndServe(ctx, mux, sub)
}
