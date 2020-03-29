package asyncp

import (
	"bytes"
	"encoding/json"
	"reflect"
)

// Event provides interface of working with message streams
type Event interface {
	// Name of the event
	Name() string

	// TaskID returns the ID of task for which it was generated
	TaskID() string

	// ForTask creates new event object to processing the task
	ForTask(task Task) Event

	// Payload returns the current message payload
	Payload() Payload

	// WithName returns new event with new name and current payload and error
	WithName(name string) Event

	// WithPayload returns new event object with extended payload context
	WithPayload(payload interface{}) Event

	// WithError returns new event object with extended error value
	WithError(err error) Event

	// SetComplete marks event as complited or no
	SetComplete(b bool)

	// IsComplete returns marker of event completion
	IsComplete() bool

	// Encode event to byte array
	Encode() ([]byte, error)

	// Decode event by the byte array
	Decode(data []byte) error
}

type idI interface {
	TaskSubID() string
}

// event structure with basic implementation of event interface
type event struct {
	notComplete bool
	name        string
	taskID      string
	taskIDs     []string
	payload     Payload
	err         error
}

// WithPayload returns new event object with payload data
func WithPayload(eventName string, data interface{}) Event {
	var (
		err     error
		payload Payload
	)
	if payload, _ = data.(Payload); payload == nil {
		payload, err = newPayload(data)
	}
	return &event{
		name:    eventName,
		payload: payload,
		err:     err,
	}
}

func (ev *event) String() string {
	data, _ := ev.Encode()
	return string(data)
}

// Copy event object
func (ev *event) Copy() *event {
	return &event{
		notComplete: ev.notComplete,
		name:        ev.name,
		taskID:      ev.taskID,
		taskIDs:     ev.taskIDs,
		payload:     ev.payload,
		err:         ev.err,
	}
}

// Name of the event
func (ev *event) Name() string {
	return ev.name
}

// TaskID returns the ID of the task which was used to generate the event
func (ev *event) TaskID() string {
	return ev.taskID
}

// ForTask creates new event object with the mark that the task has been processed
func (ev *event) ForTask(task Task) Event {
	return &event{
		name:    ev.name,
		taskID:  subidByTask(ev.TaskID(), task),
		taskIDs: append(ev.taskIDs, ev.TaskID()),
		payload: ev.payload,
	}
}

// Payload returns the current message payload
func (ev *event) Payload() Payload {
	return ev.payload
}

// Err returns error response object
func (ev *event) Err() error {
	return ev.err
}

// WithName returns new event object with new name
func (ev *event) WithName(name string) Event {
	newEvent := ev.Copy()
	newEvent.name = name
	newEvent.taskID = ""
	newEvent.taskIDs = nil
	return newEvent
}

// WithPayload returns new event object with extended payload context
func (ev *event) WithPayload(data interface{}) Event {
	newEvent := ev.Copy()
	newEvent.err = nil
	if payload, ok := data.(Payload); ok {
		newEvent.payload = payload
	} else {
		payload, err := newPayload(data)
		newEvent.payload = payload
		newEvent.err = err
	}
	return newEvent
}

// WithError returns new event object with extended error value
func (ev *event) WithError(err error) Event {
	newEvent := ev.Copy()
	newEvent.err = err
	return newEvent
}

// SetComplete marks event as complited or no
func (ev *event) SetComplete(b bool) {
	ev.notComplete = !b
}

// IsComplete returns marker of event completion
func (ev *event) IsComplete() bool {
	return !ev.notComplete
}

type encodeEvent struct {
	Name    string   `json:"name"`
	TaskID  string   `json:"task_id,omitempty"`
	TaskIDs []string `json:"task_ids,omitempty"`
	Payload []byte   `json:"payload,omitempty"`
	Err     string   `json:"error,omitempty"`
}

// Encode event to byte array
func (ev *event) Encode() ([]byte, error) {
	var (
		data []byte
		err  error
		buff bytes.Buffer
	)
	if ev.payload != nil {
		if data, err = ev.payload.Encode(); err != nil {
			return nil, err
		}
	}
	err = json.NewEncoder(&buff).Encode(&encodeEvent{
		Name:    ev.name,
		TaskID:  ev.taskID,
		TaskIDs: ev.taskIDs,
		Payload: data,
		Err:     errorString(err),
	})
	if err != nil {
		return nil, err
	}
	return buff.Bytes(), nil
}

// Decode event by the byte array
func (ev *event) Decode(data []byte) error {
	var (
		item encodeEvent
		err  = json.NewDecoder(bytes.NewBuffer(data)).Decode(&item)
	)
	if err != nil {
		return nil
	}
	ev.name = item.Name
	ev.taskID = item.TaskID
	ev.taskIDs = item.TaskIDs
	ev.payload, err = newPayload(item.Payload)
	ev.err = stringError(item.Err)
	if err != nil {
		return err
	}
	return nil
}

// UnmarshalJSON implements and wraps json.Unmarshaler interface
func (ev *event) UnmarshalJSON(data []byte) error {
	return ev.Decode(data)
}

// MarshalJSON implements and wraps json.Marshaler interface
func (ev *event) MarshalJSON() ([]byte, error) {
	return ev.Encode()
}

func subidByTask(parentID string, task Task) string {
	if task == nil {
		return parentID
	}
	if parentID != `` {
		parentID = parentID + `.`
	}
	if id, _ := task.(idI); id != nil {
		return parentID + id.TaskSubID()
	}
	return parentID + reflect.TypeOf(task).String()
}
