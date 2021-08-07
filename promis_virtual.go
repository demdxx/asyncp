package asyncp

type promiseVirtual struct {
	name            string
	targetEventName []string
}

func newPromisVirtual(name string, targetEventName ...string) *promiseVirtual {
	return &promiseVirtual{name: name, targetEventName: targetEventName}
}

// EventName accepted by the item
func (v *promiseVirtual) EventName() string { return v.name }

// TargetEventName returns name of target event
func (v *promiseVirtual) TargetEventName() []string { return v.targetEventName }

// AfterEventName map event in the event queue
func (v *promiseVirtual) AfterEventName() string { return "" }

// TargetEvent define
func (v *promiseVirtual) TargetEvent(name string) Promise {
	panic("`TargetEvent` defenition is not supported by virtual")
}

// Then execute the next task if current succeded
func (v *promiseVirtual) Then(handler interface{}) Promise {
	panic("`Then` defenition is not supported by virtual")
}

// ThenEvent which need to execute
func (v *promiseVirtual) ThenEvent(name string) { v.targetEventName = []string{name} }

// Parent promise item
func (v *promiseVirtual) Parent() Promise { return nil }

// Task executor interface
func (v *promiseVirtual) Task() Task { return nil }

// IsVirtual promise type
func (v *promiseVirtual) IsVirtual() bool { return true }
