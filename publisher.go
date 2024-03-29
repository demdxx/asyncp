package asyncp

import (
	"context"

	"github.com/geniusrabbit/notificationcenter/v2"
)

// Publisher writing interface
type Publisher = notificationcenter.Publisher

// DefaultRetranslateCount shows amount of event repeating in the pipeline
const DefaultRetranslateCount = 30

// Retranslator of the event to the stream
func Retranslator(repeatMaxCount int, pubs ...Publisher) Task {
	if repeatMaxCount <= 0 {
		repeatMaxCount = DefaultRetranslateCount
	}
	return FuncTask(func(ctx context.Context, event Event, responseWriter ResponseWriter) error {
		if _, repeats := event.Counters(); repeats > repeatMaxCount {
			return responseWriter.WriteResonse(event.WithError(ErrSkipEvent))
		}
		for _, pub := range pubs {
			if err := pub.Publish(ctx, event); err != nil {
				return err
			}
		}
		return responseWriter.WriteResonse(event)
	})
}

// Repeater send same event to the same set of pipelines
func Repeater(repeatMaxCount ...int) Task {
	maxRepears := DefaultRetranslateCount
	if len(repeatMaxCount) > 0 && repeatMaxCount[0] > 0 {
		maxRepears = repeatMaxCount[0]
	}
	return FuncTask(func(ctx context.Context, event Event, responseWriter ResponseWriter) error {
		if event.Name() == "" {
			return nil
		}
		if _, repeats := event.Counters(); repeats > maxRepears {
			return responseWriter.WriteResonse(event.WithError(ErrSkipEvent))
		}
		return responseWriter.RepeatWithResponse(event)
	})
}

type publisherEventWrapper struct {
	name string
	pub  Publisher
	mux  *TaskMux
}

// PublisherEventWrapper with fixed event name
func PublisherEventWrapper(eventName string, publisher Publisher) Publisher {
	return &publisherEventWrapper{
		name: eventName,
		pub:  publisher,
	}
}

func (wr *publisherEventWrapper) SetMux(mux *TaskMux) {
	wr.mux = mux
}

func (wr *publisherEventWrapper) Publish(ctx context.Context, messages ...any) error {
	events := make([]any, 0, len(messages))
	for _, msg := range messages {
		event := WithPayload(wr.name, msg)
		event.SetMux(wr.mux)
		events = append(events, event)
	}
	return wr.pub.Publish(ctx, events...)
}
