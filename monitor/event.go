package monitor

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// EventType defines minimal event functionality
type EventType interface {
	fmt.Stringer

	// ID of the event
	ID() uuid.UUID

	// Name of the event
	Name() string

	// Err returns error response object
	Err() error

	// CreatedAt returns the date of the event generation
	CreatedAt() time.Time
}

type errorEvent struct {
	name      string
	err       error
	createdAt time.Time
}

// NewErrorEvent returns new type event with error
func NewErrorEvent(err error) EventType {
	return &errorEvent{err: err}
}
func (ev *errorEvent) String() string       { return ev.err.Error() }
func (ev *errorEvent) ID() uuid.UUID        { return uuid.Nil }
func (ev *errorEvent) Name() string         { return ev.name }
func (ev *errorEvent) Err() error           { return ev.err }
func (ev *errorEvent) CreatedAt() time.Time { return ev.createdAt }

type wrapEvent struct {
	event EventType
	name  string
}

// WrapEventWithName returns new wrapped event
func WrapEventWithName(name string, event EventType) EventType {
	return &wrapEvent{
		event: event,
		name:  name,
	}
}

func (ev *wrapEvent) ID() uuid.UUID        { return ev.event.ID() }
func (ev *wrapEvent) String() string       { return ev.event.String() }
func (ev *wrapEvent) Name() string         { return ev.name }
func (ev *wrapEvent) Err() error           { return ev.event.Err() }
func (ev *wrapEvent) CreatedAt() time.Time { return ev.event.CreatedAt() }
