package asyncp

import (
	"context"
	"log"
	"reflect"
)

// Task describes a single execution unit
//
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
	defer func() {
		err := responseWriter.Release()
		if err != nil {
			log.Printf("release response writer: %s", err.Error())
		}
	}()
	return f(ctx, event, responseWriter)
}

// Async transforms task to the asynchronous executor
func (f FuncTask) Async(options ...AsyncOption) *AsyncTask {
	return WrapAsyncTask(f, options...)
}

// TaskFrom converts income handler type to Task interface
func TaskFrom(handler any) Task {
	switch h := handler.(type) {
	case Task:
		return h
	case func(ctx context.Context, event Event, responseWriter ResponseWriter) error:
		return FuncTask(h)
	default:
		return ExtFuncTask(h)
	}
}

var (
	errorType          = reflect.TypeOf((*error)(nil)).Elem()
	contextType        = reflect.TypeOf((*context.Context)(nil)).Elem()
	eventType          = reflect.TypeOf((*Event)(nil)).Elem()
	responseWriterType = reflect.TypeOf((*ResponseWriter)(nil)).Elem()
)

// ExtFuncTask wraps function argument with arbitrary input data type
func ExtFuncTask(f any) FuncTask {
	fv := reflect.ValueOf(f)
	if fv.Kind() != reflect.Func {
		panic("argument must be a function")
	}
	var (
		ft        = fv.Type()
		argMapper = make([]func(context.Context, Event, ResponseWriter) (reflect.Value, error), 0, ft.NumIn())
		retMapper = make([]func(reflect.Value, ResponseWriter) error, 0, ft.NumOut())
	)
	for i := 0; i < ft.NumIn(); i++ {
		inType := ft.In(i)
		switch inType {
		case contextType:
			argMapper = append(argMapper, func(ctx context.Context, _ Event, _ ResponseWriter) (reflect.Value, error) {
				return reflect.ValueOf(ctx), nil
			})
		case eventType:
			argMapper = append(argMapper, func(_ context.Context, event Event, _ ResponseWriter) (reflect.Value, error) {
				return reflect.ValueOf(event), nil
			})
		case responseWriterType:
			argMapper = append(argMapper, func(_ context.Context, _ Event, responseWriter ResponseWriter) (reflect.Value, error) {
				return reflect.ValueOf(responseWriter), nil
			})
		default:
			argMapper = append(argMapper, func(_ context.Context, event Event, _ ResponseWriter) (reflect.Value, error) {
				newValue, newValueI := newValue(inType)
				err := event.Payload().Decode(newValueI)
				return newValue, err
			})
		}
	}
	for i := 0; i < ft.NumOut(); i++ {
		outType := ft.Out(i)
		switch outType {
		case errorType:
			retMapper = append(retMapper, func(v reflect.Value, _ ResponseWriter) error {
				if v.IsNil() {
					return nil
				}
				return v.Interface().(error)
			})
		default:
			retMapper = append(retMapper, func(v reflect.Value, responseWriter ResponseWriter) error {
				if v.IsNil() {
					return nil
				}
				return responseWriter.WriteResonse(v.Interface())
			})
		}
	}
	return func(ctx context.Context, event Event, responseWriter ResponseWriter) error {
		args := make([]reflect.Value, 0, len(argMapper))
		for _, fm := range argMapper {
			arg, err := fm(ctx, event, responseWriter)
			if err != nil {
				return err
			}
			args = append(args, arg)
		}
		retVals := fv.Call(args)
		for i, fr := range retMapper {
			if err := fr(retVals[i], responseWriter); err != nil {
				return err
			}
		}
		return nil
	}
}

func newValue(t reflect.Type) (reflect.Value, any) {
	if t.Kind() == reflect.Ptr {
		return newValue(t.Elem())
	}
	v := reflect.New(t)
	i := v.Interface()
	return v, i
}
