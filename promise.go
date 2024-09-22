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
	TargetEventName() []string

	// AfterEventName map event in the event queue
	AfterEventName() string

	// TargetEvent define
	TargetEvent(name string) Promise

	// Then execute the next task if current succeeded
	Then(handler any) Promise

	// ThenEvent which need to execute
	ThenEvent(name string)

	// IsAnonymous promise type
	IsAnonymous() bool

	// Parent promise item
	Parent() Promise

	// LastPromise returns the last promise in the chain
	LastPromise() Promise

	// Task executor interface
	Task() Task

	// IsVirtual promise type
	IsVirtual() bool
}

type promise struct {
	anonymous bool

	// Accept event with name
	currentEventName string

	// Writing target name
	targetEventName []string

	// Map the task after the event
	afterEventName string

	// Parent promise object
	parent Promise

	// Parent mux object
	mux *TaskMux

	// Execution task object
	task Task
}

func newPoromise(mux *TaskMux, parent Promise, name string, task Task, anonymous bool) *promise {
	return &promise{
		currentEventName: name,
		parent:           parent,
		mux:              mux,
		task:             task,
		anonymous:        anonymous,
	}
}

func (prom *promise) EventName() string {
	return prom.currentEventName
}

func (prom *promise) TargetEventName() []string {
	if len(prom.targetEventName) == 0 {
		prProm, eventName, _ := prom.originalEventName()
		if eventName != `` && !prProm.IsVirtual() {
			// Find the target after global event which is starts from `@`
			targetNames := prom.mux.targetEventsAfter("@" + eventName)
			if len(targetNames) > 0 {
				return targetNames
			}
		}
		return prom.mux.targetEventsAfter(prom.EventName())
	}
	return prom.targetEventName
}

func (prom *promise) AfterEventName() string {
	if prom.afterEventName == `` && prom.parent != nil {
		return prom.parent.EventName()
	}
	return prom.afterEventName
}

func (prom *promise) TargetEvent(name string) Promise {
	prom.ThenEvent(name)
	return prom
}

func (prom *promise) Then(handler any) Promise {
	p := prom.mux.handleExt(prom.EventName()+">"+prom.genTargetEvent(), handler, true)
	return p
}

func (prom *promise) ThenEvent(name string) {
	prom.targetEventName = []string{name}
}

func (prom *promise) Parent() Promise {
	return prom.parent
}

func (prom *promise) LastPromise() Promise {
	for _, name := range prom.targetEventName {
		if task := prom.mux.tasks[name]; task != nil {
			return task.LastPromise()
		}
	}
	return prom
}

func (prom *promise) IsAnonymous() bool {
	return prom.anonymous
}

func (prom *promise) Origin(novirtual ...bool) (Promise, int) {
	depth := 0
	p := prom.Parent()
	if p != nil {
		if p.IsVirtual() {
			return nil, 0
		}
		for {
			depth++
			if !p.IsAnonymous() {
				break
			}
			pr := p.Parent()
			if pr == nil || pr.IsVirtual() {
				break
			}
			p = pr
		}
	}
	return p, depth
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

func (prom *promise) originalEventName() (Promise, string, int) {
	p, depth := prom.Origin()
	if p == nil {
		return nil, ``, depth
	}
	return p, p.EventName(), depth + 1
}

// generate event name after the current one
func (prom *promise) genTargetEvent() string {
	if len(prom.targetEventName) == 0 {
		_, name, depth := prom.originalEventName()
		if depth > 1 {
			name = fmt.Sprintf(`%s.%d`, name, depth)
		} else {
			name = fmt.Sprintf(`%s.1`, prom.EventName())
		}
		prom.ThenEvent(name)
	}
	return prom.targetEventName[0]
}

// IsVirtual promise type
func (prom *promise) IsVirtual() bool { return false }
