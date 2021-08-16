package asyncp

import (
	"reflect"
	"testing"

	"github.com/demdxx/asyncp/monitor"
	"github.com/stretchr/testify/assert"
)

func TestClusterAllTasks(t *testing.T) {
	cluster := Cluster{
		appInfo: &monitor.ApplicationInfo{
			Tasks: map[string][]string{
				"a":   {"a.1"},
				"a.1": {"a.2"},
				"a.2": {},
			},
		},
	}
	expectedChains := map[string][]string{"a": {"a.1", "a.2"}}
	tasks := cluster.AllTasks()
	chains := cluster.AllTaskChains()
	if !assert.True(t, reflect.DeepEqual(cluster.appInfo.Tasks, tasks), "invalid all tasks format") {
		t.Error("Expected", cluster.appInfo.Tasks, "Actual", tasks)
	}
	if !assert.True(t, reflect.DeepEqual(expectedChains, chains), "invalid all chains format") {
		t.Error("Expected", expectedChains, "Actual", chains)
	}
}
