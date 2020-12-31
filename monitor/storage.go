package monitor

import (
	"time"
)

// MetricUpdater increments data metric
type MetricUpdater interface {
	RegisterApplication(appInfo *ApplicationInfo) error
	DeregisterApplication() error
	ReceiveEvent(event EventType) error
	ExecuteTask(event EventType, execTime time.Duration) error
	ExecuteFailoverTask(event EventType, execTime time.Duration) error
}

// MetricReader of information
type MetricReader interface {
	ApplicationInfo() *ApplicationInfo
	ReceiveCount() (uint64, error)
	TaskInfo(name string) (*TaskInfo, error)
	FailoverTaskInfo(name string) (*TaskInfo, error)
}

// Storage data accessor for monitoring
type Storage interface {
	MetricUpdater
	MetricReader
}

// ClusterInfoReader provides methods of information reading
type ClusterInfoReader interface {
	ApplicationInfo() (*ApplicationInfo, error)
	TaskInfo(name string) (*TaskInfo, error)
	TaskInfoByID(id string) (*TaskInfo, error)
	ListOfNodes() (map[string][]string, int, error)
}
