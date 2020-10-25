package monitor

import "time"

// MetricUpdater increments data metric
type MetricUpdater interface {
	RegisterApplication(appInfo *ApplicationInfo) error
	DeregisterApplication() error
	IncReceiveCount() error
	IncTaskInfo(name string, execTime time.Duration, err error) error
	IncFailoverTaskInfo(execTime time.Duration, err error) error
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
	ListOfNodes() ([]string, error)
}
