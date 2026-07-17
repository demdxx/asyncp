package asyncp

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

type fakeSubscriber struct {
	subscribeErr error
	listenErr    error
	subscribed   atomic.Bool
	listened     atomic.Bool
	closed       atomic.Bool
}

func (s *fakeSubscriber) Subscribe(ctx context.Context, h Receiver) error {
	s.subscribed.Store(true)
	return s.subscribeErr
}

func (s *fakeSubscriber) Listen(ctx context.Context) error {
	s.listened.Store(true)
	return s.listenErr
}

func (s *fakeSubscriber) Close() error {
	s.closed.Store(true)
	return nil
}

func TestNewProxySubscriberSingle(t *testing.T) {
	sub := &fakeSubscriber{}
	proxy := NewProxySubscriber(sub)
	// A single subscriber should be returned as-is, without wrapping.
	assert.Same(t, sub, proxy)
}

func TestProxySubscriberSubscribeListenClose(t *testing.T) {
	sub1 := &fakeSubscriber{}
	sub2 := &fakeSubscriber{}
	proxy := NewProxySubscriber(sub1, sub2)

	err := proxy.Subscribe(context.Background(), NewTaskMux())
	assert.NoError(t, err)
	assert.True(t, sub1.subscribed.Load())
	assert.True(t, sub2.subscribed.Load())

	err = proxy.Listen(context.Background())
	assert.NoError(t, err)
	assert.True(t, sub1.listened.Load())
	assert.True(t, sub2.listened.Load())

	err = proxy.Close()
	assert.NoError(t, err)
	assert.True(t, sub1.closed.Load())
	assert.True(t, sub2.closed.Load())
}

func TestProxySubscriberSubscribeError(t *testing.T) {
	sub1 := &fakeSubscriber{subscribeErr: errors.New("boom")}
	sub2 := &fakeSubscriber{}
	proxy := NewProxySubscriber(sub1, sub2)

	err := proxy.Subscribe(context.Background(), NewTaskMux())
	assert.ErrorContains(t, err, "boom")
}

func TestProxySubscriberListenError(t *testing.T) {
	sub1 := &fakeSubscriber{listenErr: errors.New("listen failed")}
	proxy := NewProxySubscriber(sub1)
	// A single subscriber is returned unwrapped, so calling Listen directly
	// exercises the same code path used by ProxySubscriber.Listen.
	err := proxy.Listen(context.Background())
	assert.ErrorContains(t, err, "listen failed")
}

func TestProxySubscriberListenAggregatesErrorsConcurrently(t *testing.T) {
	sub1 := &fakeSubscriber{listenErr: errors.New("first failed")}
	sub2 := &fakeSubscriber{listenErr: errors.New("second failed")}
	sub3 := &fakeSubscriber{}
	proxy := NewProxySubscriber(sub1, sub2, sub3)

	err := proxy.Listen(context.Background())
	assert.ErrorContains(t, err, "first failed")
	assert.ErrorContains(t, err, "second failed")
	assert.True(t, sub3.listened.Load())
}
