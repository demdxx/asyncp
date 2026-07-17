package asyncp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPromiseVirtual(t *testing.T) {
	v := newPromisVirtual("parent", "child1", "child2")

	assert.Equal(t, "parent", v.EventName())
	assert.ElementsMatch(t, []string{"child1", "child2"}, v.TargetEventName())
	assert.Equal(t, "", v.AfterEventName())
	assert.False(t, v.IsAnonymous())
	assert.Nil(t, v.LastPromise())
	assert.Nil(t, v.Parent())
	assert.Nil(t, v.Task())
	assert.True(t, v.IsVirtual())

	v.ThenEvent("child3")
	assert.ElementsMatch(t, []string{"child3"}, v.TargetEventName())

	assert.Panics(t, func() { _ = v.TargetEvent("x") })
	assert.Panics(t, func() { _ = v.Then(testTask) })
}

func TestPromiseVirtualNoTargets(t *testing.T) {
	v := newPromisVirtual("parent")
	assert.Nil(t, v.TargetEventName())
}
