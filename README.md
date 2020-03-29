# Asyncp processing framework

[![Build Status](https://travis-ci.org/demdxx/asyncp.svg?branch=master)](https://travis-ci.org/demdxx/asyncp)
[![Go Report Card](https://goreportcard.com/badge/github.com/demdxx/asyncp)](https://goreportcard.com/report/github.com/demdxx/asyncp)
[![GoDoc](https://godoc.org/github.com/demdxx/asyncp?status.svg)](https://godoc.org/github.com/demdxx/asyncp)
[![Coverage Status](https://coveralls.io/repos/github/demdxx/asyncp/badge.svg)](https://coveralls.io/github/demdxx/asyncp)

> License Apache 2.0

The simple framework to build async task execution programs.

## Pipelines

Sequentially apply a list of tasks. Pipelines allow to split one big
processing part for small simple steps to simplify the complex logic.

The pipeline is atomic, all tasks in the pipeline will successfully
execute the whole task or all not.

## Example program

```go
import (
  ...
  "github.com/demdxx/asyncp"
  ...
)

func main() {
  taskQueueSub := nats.NewSubscriber(...)
  taskQueuePub := nats.NewPublisher(...)

  // Create new async multiplexer
  mx := asyncp.NewTaskMux(
    // Define default strem message queue
    asyncp.WithStreamResponseFactory(taskQueuePub),
    asyncp.WithPanicHandler(...),
    asyncp.WithErrorHandler(...),
  )

  // Create new task handler to download articles by RSS
  mx.Handle("rss", asyncp.FuncTask(downloadRSSList)).
    Then(asyncp.FuncTask(downloadRSSItem)).
    Then(asyncp.FuncTask(updateRSSArticles))

  // Create new task handler to process video files
  mx.Handle("video", asyncp.FuncTask(loadVideoForProcessing)).
    Then(asyncp.FuncTask(makeVideoThumbs)).
    Then(asyncp.FuncTask(convertVideoFormat))

  // Send report to user (event contains login and email target)
  mx.Handle("email", pipeline.New(
    `userinfo`, asyncp.FuncTask(assembleBasicInfo),
    `changes`, asyncp.FuncTask(assembleAllChangesForUser),
    `template`, asyncp.FuncTask(assembleEmailHTMLTemplate),
    pipeline.New(
      asyncp.FuncTask(sendNotification),
      asyncp.FuncTask(sendSendLogs),
    ),
  )).Then(sendEmailTask)

  // Retranslate all message to the queue if can`t process
  mx.Failver(asyncp.Retranslator(taskQueuePub))

  taskQueueSub.Subscribe(context.Background(), mx)
  taskQueueSub.Listen(context.Background())
}
```
