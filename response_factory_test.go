package asyncp

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testPublisher struct{ name string }

func (p *testPublisher) String() string                                     { return p.name }
func (p *testPublisher) Publish(ctx context.Context, messages ...any) error { return nil }
func (p *testPublisher) PublishAndReturnIDs(ctx context.Context, messages ...any) ([]string, error) {
	return nil, nil
}

func TestMultistreamResponseFactory(t *testing.T) {
	fc := NewMultistreamResponseFactory(
		"item1", "item2", &testPublisher{name: "test1"},
		regexp.MustCompile(`item\d+`), &testPublisher{name: "test2"},
		&testPublisher{name: "test3"},
	).(*mutistreamResponseFactory)

	pub := fc.publisher(WithPayload("item2", nil))
	assert.Equal(t, "test1", pub.(fmt.Stringer).String())

	pub = fc.publisher(WithPayload("item3", nil))
	assert.Equal(t, "test2", pub.(fmt.Stringer).String())

	pub = fc.publisher(WithPayload("something", nil))
	assert.Equal(t, "test3", pub.(fmt.Stringer).String())

	assert.Panics(t, func() { NewMultistreamResponseFactory(100) })
}
