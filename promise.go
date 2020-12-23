package asyncp

import (
	"fmt"
	"io"
)

// Promise describe the behaviour of Single task item
type Promise interface {
	// EventName accepted by the item
	EventName() string

	// TargetEventName returns name of target event
	TargetEventName() string

	// TargetEvent define
	TargetEvent(name string) Promise

	// Then execute the next task if current succeded
	Then(handler interface{}) Promise

	// ThenEvent which need to execute
	ThenEvent(name string)

	// Parent promise item
	Parent() Promise

	// Task executor interface
	Task() Task
}

type promise struct {
	// Accept event with name
	currentEventName string

	// Writing target name
	targetEventName string

	// Parent promise object
	parent Promise

	// Parent mux object
	mux *TaskMux

	// Execution task object
	task Task
}

func newPoromise(mux *TaskMux, parent Promise, name string, task Task) *promise {
	return &promise{
		currentEventName: name,
		parent:           parent,
		mux:              mux,
		task:             task,
	}
}

func (prom *promise) EventName() string {
	return prom.currentEventName
}

func (prom *promise) TargetEventName() string {
	return prom.targetEventName
}

func (prom *promise) TargetEvent(name string) Promise {
	prom.targetEventName = name
	return prom
}

func (prom *promise) Then(handler interface{}) Promise {
	p := prom.mux.Handle(prom.genTargetEvent(), TaskFrom(handler))
	p.(*promise).parent = prom
	return p
}

func (prom *promise) ThenEvent(name string) {
	prom.targetEventName = name
}

func (prom *promise) Parent() Promise {
	return prom.parent
}

func (prom *promise) Task() Task {
	return prom.task
}

func (prom *promise) Close() error {
	if closer, _ := prom.task.(io.Closer); closer != nil {
		return closer.Close()
	}
	return nil
}

func (prom *promise) originalEventName() (_ string, depth int) {
	p := prom.Parent()
	for ; p != nil; p = p.Parent() {
		depth++
		if p.Parent() == nil {
			break
		}
	}
	if p == nil {
		return ``, depth
	}
	return p.EventName(), depth + 1
}

func (prom *promise) genTargetEvent() string {
	if prom.targetEventName == `` {
		name, depth := prom.originalEventName()
		if depth > 1 {
			prom.targetEventName = fmt.Sprintf(`%s.%d`, name, depth)
		} else {
			prom.targetEventName = fmt.Sprintf(`%s.%d`, prom.EventName(), 1)
		}
	}
	return prom.targetEventName
}
