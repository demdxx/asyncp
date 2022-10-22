package asyncp

import (
	"context"
	"log"

	"github.com/demdxx/rpool/v2"
)

// AsyncOption type options tune
type AsyncOption func(opt *AsyncOptions)

// AsyncOptions contains concurrent execution pool
type AsyncOptions struct {
	// pool options
	poolOptions []rpool.Option
}

// Pool of execution
func (opt *AsyncOptions) Pool(fnk func(any)) *rpool.PoolFunc[any] {
	return rpool.NewPoolFunc(fnk, opt.poolOptions...)
}

// WithWorkerCount change count of workers
func WithWorkerCount(count int) AsyncOption {
	return func(opt *AsyncOptions) {
		opt.poolOptions = append(opt.poolOptions, rpool.WithWorkerCount(count))
	}
}

// WithWorkerPoolSize setup maximal size of worker pool
func WithWorkerPoolSize(size int) AsyncOption {
	return func(opt *AsyncOptions) {
		opt.poolOptions = append(opt.poolOptions, rpool.WithWorkerPoolSize(size))
	}
}

// WithRecoverHandler defined error handler
func WithRecoverHandler(f func(any)) AsyncOption {
	return func(opt *AsyncOptions) {
		opt.poolOptions = append(opt.poolOptions, rpool.WithRecoverHandler(f))
	}
}

type asyncTaskParams struct {
	ctx   context.Context
	event Event
	rw    ResponseWriter
}

// AsyncTask processor
type AsyncTask struct {
	execPool *rpool.PoolFunc[any]
	task     Task
}

// WrapAsyncTask as async executor
func WrapAsyncTask(task Task, options ...AsyncOption) *AsyncTask {
	var opts AsyncOptions
	for _, opt := range options {
		opt(&opts)
	}
	asyncTask := &AsyncTask{task: task}
	asyncTask.execPool = opts.Pool(asyncTask.handler)
	return asyncTask
}

// Execute the list of subtasks with input data collection.
func (t *AsyncTask) Execute(ctx context.Context, event Event, responseWriter ResponseWriter) error {
	t.execPool.Call(&asyncTaskParams{ctx: ctx, event: event, rw: responseWriter})
	return nil
}

func (t *AsyncTask) handler(ctx any) {
	p := ctx.(*asyncTaskParams)
	defer func() {
		err := p.rw.Release()
		if err != nil {
			log.Printf("release response writer: %s", err.Error())
		}
	}()
	err := t.task.Execute(p.ctx, p.event, p.rw)
	if err != nil {
		panic(err)
	}
}

// Close execution pool and finish handler processing
func (t *AsyncTask) Close() error {
	return t.execPool.Close()
}
