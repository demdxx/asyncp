package asyncp

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/geniusrabbit/notificationcenter/v2"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
)

// Error list...
var (
	ErrChanelTaken = errors.New(`chanel has been taken`)
)

// Stream writing interface
type Stream = notificationcenter.Publisher

// TaskMux object which controls the workflow of task execution
type TaskMux struct {
	cluster ClusterExt

	// Chanel name + task with responser
	tasks map[string]Promise

	// Maps final task of the chanel with tasks from other chanels or clusters.
	// All linked external events starts from `@`; @globalEvent -> targetEvent
	hiddenTaskMapping map[string][]string

	// Default task if not found
	failoverTask Promise

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

	// EventAllocator provides interface of event object management
	eventAllocator EventAllocator
}

// NewTaskMux server object
func NewTaskMux(options ...Option) *TaskMux {
	var opts Options
	for _, opt := range options {
		opt(&opts)
	}
	mux := &TaskMux{
		tasks:             map[string]Promise{},
		hiddenTaskMapping: map[string][]string{},
		panicHandler:      opts.PanicHandler,
		errorHandler:      opts.ErrorHandler,
		mainExecContext:   opts.MainExecContext,
		contextWrapper:    opts.ContextWrapper,
		responseFactory:   opts.ResponseFactory,
		cluster:           opts.Cluster,
		eventAllocator:    opts._eventAllocator(),
	}
	if muxSet, ok := mux.responseFactory.(interface{ SetMux(mux *TaskMux) }); ok {
		muxSet.SetMux(mux)
	}
	return mux
}

// Handle register new task for specific chanel
// Task after other task can be defined by "parentTaskName>currentTaskName"
func (srv *TaskMux) Handle(taskName string, handler any) Promise {
	return srv.handleExt(taskName, handler, false)
}

func (srv *TaskMux) handleExt(name string, handler any, anonymous bool) Promise {
	var (
		parentPromis             Promise
		parentTaskName, taskName = prepareTaskName(name)
	)
	if srv.tasks == nil {
		srv.tasks = map[string]Promise{}
	}
	if _, ok := srv.tasks[taskName]; ok {
		panic(errors.Wrap(ErrChanelTaken, taskName))
	}
	if parentTaskName != "" {
		// If there is no parent promis in the scope of local tasks
		// then the parent is external task
		if parentPromis = srv.tasks[parentTaskName]; parentPromis != nil {
			parentPromis = parentPromis.LastPromise()
		}
	}

	taskItemValue := newPoromise(srv, parentPromis, taskName, TaskFrom(handler), anonymous)
	srv.tasks[taskName] = taskItemValue

	if parentTaskName != "" && parentPromis == nil {
		// Links global event name and the target external one
		taskItemValue.parent = newPromisVirtual(parentTaskName, taskName)
		parentTaskName = "@" + parentTaskName
		srv.hiddenTaskMapping[parentTaskName] = append(srv.hiddenTaskMapping[parentTaskName], taskName)
	}

	// If parent task is not virtual then add event to the parent task
	if parentPromis != nil && !parentPromis.IsVirtual() {
		parentPromis.ThenEvent(taskName)
	}
	return taskItemValue
}

// Failover handler if was reseaved event with unsappoted event
func (srv *TaskMux) Failover(task any) error {
	srv.failoverTask = &promise{task: TaskFrom(task)}
	return nil
}

// Receive definds the processing function
func (srv *TaskMux) Receive(msg Message) error {
	event, err := srv.eventAllocator.Decode(msg)
	if event != nil {
		defer func() {
			_ = srv.eventAllocator.Release(event)
			if srv.panicHandler != nil {
				if err := recover(); err != nil {
					srv.panicHandler(nil, event, err)
				}
			}
		}()
	}
	if srv.cluster != nil {
		_ = srv.cluster.ReceiveEvent(event, err)
	}
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
	event.SetPromise(task)
	event.SetMux(srv)

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
				if srv.cluster != nil {
					_ = srv.cluster.ExecEvent(isFailover, event, time.Since(startTime), err.(error))
				}
			}
		}()
	}

	ctx := srv.newExecContext()
	wrt := srv.borrowResponseWriter(ctx, task, event)

	// Execute the task
	err := task.Task().Execute(ctx, event, wrt)
	if srv.cluster != nil {
		_ = srv.cluster.ExecEvent(isFailover, event, time.Since(startTime), err)
	}

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
	if srv.cluster != nil {
		return srv.cluster.RegisterApplication(
			srv.newExecContext(), srv)
	}
	return nil
}

// Close task schedule and all subtasks
func (srv *TaskMux) Close() error {
	if srv.cluster != nil {
		_ = srv.cluster.UnregisterApplication()
	}
	var err error
	if srv == nil || srv.tasks == nil {
		return nil
	}
	for _, promise := range srv.tasks {
		if closer, ok := promise.(io.Closer); ok {
			err = multierr.Append(err, closer.Close())
		}
	}
	return err
}

// CompleteTasks checks the event completion state
func (srv *TaskMux) CompleteTasks(event Event) (totalTasks, completedTasks []string) {
	var tasks map[string][]string
	if srv.cluster != nil {
		tasks = srv.cluster.AllTasks()
	} else {
		tasks = srv.TaskMap()
	}
	doneTasks := append(event.DoneTasks(), event.Name())
	allPossibleTasks := map[string]bool{}
	for _, taskName := range doneTasks {
		for _, name := range tasks[taskName] {
			allPossibleTasks[name] = true
		}
		for _, name := range tasks["@"+taskName] {
			allPossibleTasks[name] = true
		}
		if tasks[taskName] != nil || tasks["@"+taskName] != nil {
			allPossibleTasks[taskName] = true
		}
	}
	for taskName := range allPossibleTasks {
		for _, name := range tasks[taskName] {
			allPossibleTasks[name] = true
		}
	}
	totalTasks = make([]string, 0, len(allPossibleTasks))
	for tname := range allPossibleTasks {
		totalTasks = append(totalTasks, tname)
	}
	return totalTasks, completedTasks
}

func (srv *TaskMux) borrowResponseWriter(ctx context.Context, prom Promise, event Event) ResponseWriter {
	if srv.responseFactory == nil {
		return &responseProxyWriter{mux: srv, event: event, promise: prom}
	}
	return srv.responseFactory.Borrow(ctx, prom, event)
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

func (srv *TaskMux) targetEventsAfter(eventName string) []string {
	if srv.hiddenTaskMapping != nil && len(srv.hiddenTaskMapping[eventName]) > 0 {
		return srv.hiddenTaskMapping[eventName]
	}
	if srv.cluster != nil {
		return srv.cluster.TargetEventsAfter(eventName)
	}
	return nil
}

// TaskMap returns linked list of events
func (srv *TaskMux) TaskMap() map[string][]string {
	mp := make(map[string][]string, len(srv.tasks)+len(srv.hiddenTaskMapping))
	if srv.tasks != nil {
		for eventName, promiseObject := range srv.tasks {
			mp[eventName] = mergeStrArr(mp[eventName], promiseObject.TargetEventName())
		}
	}

	if srv.hiddenTaskMapping != nil {
		for eventName, targetEvent := range srv.hiddenTaskMapping {
			mp[eventName] = mergeStrArr(mp[eventName], targetEvent)
		}
	}

	return mp
}

func prepareTaskName(name string) (parent, target string) {
	splitName := strings.SplitN(name, ">", 2)
	if len(splitName) > 1 {
		parent = splitName[0]
		target = splitName[1]
	} else {
		target = splitName[0]
	}
	return parent, target
}

var _ = (notificationcenter.Receiver)((*TaskMux)(nil))
