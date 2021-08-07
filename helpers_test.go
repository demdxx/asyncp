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

func TestMapSize(t *testing.T) {
	assert.Equal(t, 0, taskMapSize(nil))
	assert.Equal(t, 0, eventMapSize(nil))
	assert.Equal(t, 0, taskMapSize(map[string]Promise{}))
	assert.Equal(t, 0, eventMapSize(map[string][]string{}))
	assert.Equal(t, 1, taskMapSize(map[string]Promise{"": nil}))
	assert.Equal(t, 1, eventMapSize(map[string][]string{"": nil}))
}
