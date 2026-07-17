# Asyncp processing framework

[![Build Status](https://github.com/demdxx/asyncp/workflows/run%20tests/badge.svg)](https://github.com/demdxx/asyncp/actions?workflow=run%20tests)
[![Coverage Status](https://coveralls.io/repos/github/demdxx/asyncp/badge.svg?branch=master)](https://coveralls.io/github/demdxx/asyncp?branch=master)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/demdxx/asyncp/v2)](https://pkg.go.dev/github.com/demdxx/asyncp/v2)

> License Apache 2.0

The simple framework to build async task execution programs.

- Chain handlers into pipelines with automatic event routing between steps
- Run tasks synchronously or offload them to a worker pool via `Async()`
- Publish/subscribe over pluggable transports: in-memory (`gochan`), NATS, Kafka, Redis
- Scale horizontally in cluster mode, with live state in `apmonitor`

## Installation

```bash
go get github.com/demdxx/asyncp/v2
```

Requires Go 1.25 or newer (see `go.mod`).

## Pipelines

Sequentially apply a list of tasks. Pipelines allow to split one big
processing part for small simple steps to simplify the complex logic.

The pipeline is atomic, all tasks in the pipeline will successfully
execute the whole task or all not.

## Example program

```go
import (
  ...
  "github.com/demdxx/asyncp/v2"
  "github.com/demdxx/asyncp/v2/streams"
  ...
)

func main() {
  taskQueueSub := nats.NewSubscriber(...)
  taskQueuePub := nats.NewPublisher(...)

  // Create new async multiplexer
  mx := asyncp.NewTaskMux(
    // Define default stream message queue
    asyncp.WithStreamResponsePublisher(taskQueuePub),
    asyncp.WithPanicHandler(...),
    asyncp.WithErrorHandler(...),
    asyncp.WithCluster(...),
  )
  defer func() { _ = mx.Close() }()

  // Create new task handler to download articles by RSS
  mx.Handle("rss", downloadRSSList).
    Then(downloadRSSItem).
    Then(updateRSSArticles)

  // Create new task handler to process video files
  mx.Handle("video", loadVideoForProcessing).
    Then(makeVideoThumbs).
    Then(convertVideoFormat)

  // Send report to user (event contains login and email target)
  mx.Handle("email", pipeline.New(
    `userinfo`, assembleBasicInfo,
    `changes`, assembleAllChangesForUser,
    `template`, assembleEmailHTMLTemplate,
    pipeline.New(
      sendNotification,
      sendSendLogs,
    ),
  )).Then(sendEmailTask)

  // Retranslate all message to the queue if can`t process
  mx.Failover(asyncp.Retranslator(0, taskQueuePub))

  // Alternative:
  // taskQueueSub.Subscribe(context.Background(), mx)
  // taskQueueSub.Listen(context.Background())
  err = streams.ListenAndServe(context.Background(), mx,
    taskQueueSub, "nats://host:2222/group?topics=topicName")
}
```

Convert task to async executor.

```go
atask := asyncp.WrapAsyncTask(task,
  WithWorkerCount(10),
  WithWorkerPoolSize(20),
  WithRecoverHandler(func(rec any) {
    // ...
  }))

// Or

asyncp.FuncTask(assembleBasicInfo).Async()
```

## Cluster mode

The framework supports cluster task processing.
Some of the servers can execute some specific tasks like video processing or windows specific stuff.

To extend cluster base functionality need to create in other applications (with the same sync options)
linked task `baseHandlerTaskName>myNewClusterTask`.

```go
func main() {
  ...
  // After RSS parsing we need to prcess video files from links if it's present
  mx.Handle("rss>videoExtraction", loadVideoForProcessing).
    Then(makeVideoThumbs).
    Then(convertVideoFormat)
  ...
}
```

## Apmonitor tool

Displays state of the cluster and every task common state.

![apmonitor tool](docs/apmonitor.png "Apmonitor")

## Streams / transports

The [`streams`](streams) package wires a `TaskMux` to a message transport via
`streams.ListenAndServe`. Supported backends:

- `gochan` — in-memory buffered channel, useful for tests and single-process demos
- `nats` — NATS / NATS JetStream
- `kafka` — Apache Kafka
- `redis` — Redis streams

## Examples

Runnable demos live under [`example`](example):

- [`example/counter`](example/counter) — periodically increments a counter and
  reports processing metrics to a Redis-backed cluster monitor
- [`example/rss-reader`](example/rss-reader) — downloads an RSS feed, fetches
  every linked item concurrently through a pipeline, and prints a summary

## Development

```bash
make test  # go test -race ./...
make lint  # golangci-lint run ./...
```
