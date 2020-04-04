package asyncp

import "context"

// Task describes a single execution unit
//go:generate mockgen -source $GOFILE -package mocks -destination mocks/task.go
type Task interface {
	// Execute the list of subtasks with input data collection.
	// It returns the new data collection which will be used in the next tasks as input params.
	Execute(ctx context.Context, event Event, responseWriter ResponseWriter) error
}

// FuncTask provides implementation of Task interface for function pointer
type FuncTask func(ctx context.Context, event Event, responseWriter ResponseWriter) error

// Execute the list of subtasks with input data collection.
func (f FuncTask) Execute(ctx context.Context, event Event, responseWriter ResponseWriter) error {
	return f(ctx, event, responseWriter)
}

// Async transforms task to the asynchronous executor
func (f FuncTask) Async(options ...AsyncOption) *AsyncTask {
	return WrapAsyncTask(f, options...)
}
