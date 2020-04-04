package asyncp

import (
	"context"

	"github.com/geniusrabbit/notificationcenter"
	"github.com/pkg/errors"
)

// Error list...
var (
	ErrChanelTaken = errors.New(`chanel has been taken`)
)

// Stream writing interface
type Stream = notificationcenter.Publisher

// TaskMux object which controls the workflow of task execution
type TaskMux struct {
	// Chanel name + task with responser
	tasks map[string]*promise

	// Default task if not found
	failoverTask *promise

	// errorHandler process panic responses
	panicHandler func(Task, Event, interface{})

	// errorHandler process error responses
	errorHandler func(Task, Event, error)

	// Allocate new specific writer for every event separately
	responseFactory ResponseWriterFactory
}

// NewTaskMux server object
func NewTaskMux(options ...Option) *TaskMux {
	var opts Options
	for _, opt := range options {
		opt(&opts)
	}
	return &TaskMux{
		tasks:           map[string]*promise{},
		panicHandler:    opts.PanicHandler,
		errorHandler:    opts.ErrorHandler,
		responseFactory: opts.StreamResponseFactory,
	}
}

// Handle register new task for specific chanel
func (srv *TaskMux) Handle(chanelName string, task Task) Promise {
	if srv.tasks == nil {
		srv.tasks = map[string]*promise{}
	}
	if _, ok := srv.tasks[chanelName]; ok {
		panic(errors.Wrap(ErrChanelTaken, chanelName))
	}
	taskItemValue := newPoromise(srv, nil, chanelName, task)
	srv.tasks[chanelName] = taskItemValue
	return taskItemValue
}

// Failver handler if was reseaved event with unsappoted event
func (srv *TaskMux) Failver(task Task) error {
	srv.failoverTask = &promise{task: task}
	return nil
}

// Receive definds the processing function
func (srv *TaskMux) Receive(msg Message) error {
	event, err := srv.eventDecode(msg.Body())
	if err != nil {
		return err
	}
	task, ok := srv.tasks[event.Name()]
	if !ok {
		task = srv.failoverTask
	}
	if task == nil {
		return nil
	}

	// process task panics
	if srv.panicHandler != nil {
		defer func() {
			if err := recover(); err != nil {
				srv.panicHandler(task.Task(), event, err)
			}
		}()
	}

	wrt := srv.borrowResponseWriter(task, event)
	defer srv.releaseResponseWriter(wrt)

	// Execute the task
	ctx := context.Background()
	err = task.task.Execute(ctx, event, wrt)

	if err != nil {
		if srv.errorHandler != nil {
			srv.errorHandler(task.Task(), event, err)
		} else {
			return err
		}
	}
	return msg.Ack()
}

// Close task schedule and all subtasks
func (srv *TaskMux) Close() error {
	var err multiError
	if srv == nil || srv.tasks == nil {
		return nil
	}
	for _, promise := range srv.tasks {
		err.Add(promise.Close())
	}
	return err.AsError()
}

func (srv *TaskMux) borrowResponseWriter(prom *promise, event Event) ResponseWriter {
	if srv.responseFactory == nil {
		return &responseProxyWriter{mux: srv, parent: event, promise: prom}
	}
	return srv.responseFactory.Borrow(prom, event)
}

func (srv *TaskMux) releaseResponseWriter(rw ResponseWriter) {
	if srv.responseFactory == nil {
		return
	}
	srv.responseFactory.Release(rw)
}

func (srv *TaskMux) eventDecode(data []byte) (*event, error) {
	event := &event{}
	return event, event.Decode(data)
}
