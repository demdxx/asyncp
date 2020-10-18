package monitor

import "time"

// Storage data accessor for monitoring
type Storage interface {
	ApplicationInfo() *ApplicationInfo
	RegisterApplication(appInfo *ApplicationInfo) error
	DeregisterApplication() error
	ReceiveCount() (uint64, error)
	IncReceiveCount() error
	TaskInfo(name string) (*TaskInfo, error)
	IncTaskInfo(name string, execTime time.Duration, err error) error
	FailoverTaskInfo(name string) (*TaskInfo, error)
	IncFailoverTaskInfo(execTime time.Duration, err error) error
}

// ClusterInfoReader provides methods of information reading
type ClusterInfoReader interface {
	ApplicationInfo() (*ApplicationInfo, error)
	TaskInfo(name string) (*TaskInfo, error)
	ListOfNodes() ([]string, error)
}
