package asyncp

import (
	"context"
	"fmt"
	"time"

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
	// monitor accessor
	monitor *Monotor

	// Chanel name + task with responser
	tasks map[string]*promise

	// Default task if not found
	failoverTask *promise

	// mainExecContext as default for any execution request
	mainExecContext context.Context

	// errorHandler process panic responses
	panicHandler PanicHandlerFnk

	// errorHandler process error responses
	errorHandler ErrorHandlerFnk

	// contextWrapper for execution context preparation
	contextWrapper ContextWrapperFnk

	// Allocate new specific writer for every event separately
	responseFactory ResponseWriterFactory
}

// NewTaskMux server object
func NewTaskMux(options ...Option) *TaskMux {
	var opts Options
	for _, opt := range options {
		opt(&opts)
	}
	mux := &TaskMux{
		tasks:           map[string]*promise{},
		panicHandler:    opts.PanicHandler,
		errorHandler:    opts.ErrorHandler,
		contextWrapper:  opts.ContextWrapper,
		responseFactory: opts.ResponseFactory,
		monitor:         opts.Monitor,
	}
	if muxSet, ok := mux.responseFactory.(interface{ SetMux(mux *TaskMux) }); ok {
		muxSet.SetMux(mux)
	}
	return mux
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
	srv.monitor.receiveEvent(event, err)
	if err != nil {
		return err
	}
	if err = srv.ExecuteEvent(event); err != nil {
		return err
	}
	return msg.Ack()
}

// ExecuteEvent with mux executor
func (srv *TaskMux) ExecuteEvent(event Event) error {
	task, ok := srv.tasks[event.Name()]
	isFailover := false
	if !ok {
		isFailover = true
		task = srv.failoverTask
	}
	if task == nil {
		return nil
	}

	startTime := time.Now()

	// process task panics
	if srv.panicHandler != nil {
		defer func() {
			if err := recover(); err != nil {
				srv.panicHandler(task.Task(), event, err)
				switch err.(type) {
				case error:
				default:
					err = fmt.Errorf("%v", err)
				}
				srv.monitor.execEvent(isFailover, event, time.Since(startTime), err.(error))
			}
		}()
	}

	ctx := srv.newExecContext()
	wrt := srv.borrowResponseWriter(ctx, task, event)

	// Execute the task
	err := task.task.Execute(ctx, event, wrt)
	srv.monitor.execEvent(isFailover, event, time.Since(startTime), err)

	if err != nil {
		if srv.errorHandler != nil {
			srv.errorHandler(task.Task(), event, err)
		} else {
			return err
		}
	}
	return nil
}

// FinishInit of the task server
func (srv *TaskMux) FinishInit() error {
	return srv.monitor.register(srv)
}

// Close task schedule and all subtasks
func (srv *TaskMux) Close() error {
	srv.monitor.deregister()
	var err multiError
	if srv == nil || srv.tasks == nil {
		return nil
	}
	for _, promise := range srv.tasks {
		err.Add(promise.Close())
	}
	return err.AsError()
}

func (srv *TaskMux) borrowResponseWriter(ctx context.Context, prom *promise, event Event) ResponseWriter {
	if srv.responseFactory == nil {
		return &responseProxyWriter{mux: srv, event: event, promise: prom}
	}
	return srv.responseFactory.Borrow(ctx, prom, event)
}

func (srv *TaskMux) eventDecode(data []byte) (*event, error) {
	event := &event{}
	return event, event.Decode(data)
}

func (srv *TaskMux) newExecContext() context.Context {
	ctx := srv.mainExecContext
	if ctx == nil {
		ctx = context.Background()
	}
	if srv.contextWrapper != nil {
		ctx = srv.contextWrapper(ctx)
	}
	return ctx
}

// EventMap returns linked list of events
func (srv *TaskMux) EventMap() map[string]string {
	if srv.tasks == nil {
		return map[string]string{}
	}
	mp := make(map[string]string, len(srv.tasks))
	for eventName, promiseObject := range srv.tasks {
		mp[eventName] = promiseObject.TargetEventName()
	}
	return mp
}
