package asyncp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringArrMerge(t *testing.T) {
	assert.ElementsMatch(t, []string{}, mergeStrArr(nil))
	assert.ElementsMatch(t, []string{}, mergeStrArr(nil, nil))
	assert.ElementsMatch(t, []string{}, mergeStrArr([]string{}, []string{}))
	assert.ElementsMatch(t, []string{"a", "b"}, mergeStrArr([]string{"a", "b"}))
	assert.ElementsMatch(t, []string{"a", "b"}, mergeStrArr([]string{"a"}, []string{"b"}))
	assert.ElementsMatch(t, []string{"a", "b"}, mergeStrArr([]string{"a", "a"}, []string{"b", "b"}))
	assert.ElementsMatch(t, []string{"a", "b"}, mergeStrArr([]string{"a", "b"}, []string{"a", "b"}))
}

func TestStringArrExclude(t *testing.T) {
	assert.ElementsMatch(t, []string{}, excludeFromStrArr(nil))
	assert.ElementsMatch(t, []string{}, excludeFromStrArr([]string{}))
	assert.ElementsMatch(t, []string{}, excludeFromStrArr([]string{}, "a", "b"))
	assert.ElementsMatch(t, []string{}, excludeFromStrArr([]string{"a", "b"}, "a", "b"))
	assert.ElementsMatch(t, []string{"a"}, excludeFromStrArr([]string{"a", "b"}, "b"))
	assert.ElementsMatch(t, []string{"a", "b"}, excludeFromStrArr([]string{"a", "b"}, "c", "d"))
}
