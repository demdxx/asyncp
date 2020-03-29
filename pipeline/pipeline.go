package pipeline

import (
	"context"
	"fmt"

	"github.com/demdxx/asyncp"
	"github.com/pkg/errors"
)

// Error list...
var (
	ErrInvalidInputParams      = errors.New(`invalid input parameter`)
	ErrInvalidInputParamsOrder = errors.New(`invalid input parameter order`)
	ErrTaskIsRegistered        = errors.New(`task is registered`)
	ErrUndefinedTask           = errors.New(`task is undefined`)
)

type item struct {
	name string
	task asyncp.Task
}

// Pipeline implements interface of the Pipeline interface
type Pipeline struct {
	tasks []*item
}

// New defines the pipeline execution object of task sequences
//
// Example:
//   pipe := pipeline.New("meta", NewMetaExtractor(), NewSummarise())
//   data, err = pipe.Execute(ctx, data)
func New(tasks ...interface{}) *Pipeline {
	var (
		pipe = &Pipeline{}
		name string
	)
	for _, v := range tasks {
		switch vl := v.(type) {
		case asyncp.Task:
			if err := pipe.AddTask(name, vl); err != nil {
				panic(err)
			}
			name = ""
		case string:
			if name != "" {
				panic(ErrInvalidInputParamsOrder)
			}
			name = vl
		default:
			panic(ErrInvalidInputParams)
		}
	}
	return pipe
}

// AddTask puts new task in the list
func (p *Pipeline) AddTask(name string, task asyncp.Task) error {
	if name != "" {
		if task, _ := p.TaskByName(name); task != nil {
			return errors.Wrap(ErrTaskIsRegistered, name)
		}
	} else {
		name = fmt.Sprintf("task%d", len(p.tasks)+1)
	}
	p.tasks = append(p.tasks, &item{name: name, task: task})
	return nil
}

// Execute the list of subtasks with input data collection.
// This is the sequence of subtasks which executes in the order of definition of tasks.
// It returns the new data collection which will be used in the next tasks as input params.
func (p *Pipeline) Execute(ctx context.Context, event asyncp.Event, responseWriter asyncp.ResponseWriter) (err error) {
	var (
		streamWriter                       = newStream(event)
		streamReader                       = newStream(event)
		rwriter      asyncp.ResponseWriter = streamWriter
	)
	defer func() {
		_ = streamWriter.Close()
		_ = streamReader.Close()
	}()
	if err = streamReader.WriteResonse(event); err != nil {
		return err
	}
	for i, task := range p.tasks {
		if i == len(p.tasks)-1 {
			rwriter = responseWriter
		}
		ev := streamReader.nextEvent()
		for ; ev != nil; ev = streamReader.nextEvent() {
			if err := task.task.Execute(ctx, ev, rwriter); err != nil {
				if err = responseWriter.WriteResonse(ev.WithError(err)); err != nil {
					return err
				}
			}
		}
		streamWriter, streamReader = streamReader, streamWriter
		streamWriter.Reset()
		rwriter = streamWriter
	}
	return nil
}

// TaskByName returns stored task by name
func (p *Pipeline) TaskByName(name string) (asyncp.Task, error) {
	for _, task := range p.tasks {
		if task.name == name {
			return task.task, nil
		}
	}
	return nil, ErrUndefinedTask
}

// TaskByIndex returns stored task by index in the list
func (p *Pipeline) TaskByIndex(idx int) (asyncp.Task, error) {
	if idx >= 0 || idx < len(p.tasks) {
		return p.tasks[idx].task, nil
	}
	return nil, ErrUndefinedTask
}
