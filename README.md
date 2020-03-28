# Asyncp processing framework

The simple framework to build async task execution programs.

## Example programm

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

  // Retranslate all message to the queue if can`t process
  mx.Failver(asyncp.Retranslator(taskQueuePub))

  taskQueueSub.Subscribe(context.Background(), mx)
  taskQueueSub.Listen(context.Background())
}
```
