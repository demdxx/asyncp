package redis

import (
	"fmt"
	"strings"
	"sync"

	"github.com/demdxx/asyncp/monitor"
	"github.com/demdxx/gocast"
	"github.com/go-redis/redis"
)

// ClusterInfoReader overrides group node accessing
type ClusterInfoReader struct {
	mx sync.Mutex

	appID   []string
	appName string
	client  redis.Cmdable
}

// NewClusterInfoReader interface implementation
func NewClusterInfoReader(client redis.Cmdable, appName string) *ClusterInfoReader {
	return &ClusterInfoReader{
		appName: appName,
		client:  client,
	}
}

// NewClusterInfoReaderByURL from URL
func NewClusterInfoReaderByURL(connectURL, appName string) (*ClusterInfoReader, error) {
	redisClient, err := connectRedis(connectURL)
	if err != nil {
		return nil, err
	}
	return NewClusterInfoReader(redisClient, appName), nil
}

// ApplicationInfo returns application information
func (s *ClusterInfoReader) ApplicationInfo() (*monitor.ApplicationInfo, error) {
	appList, err := s.ListOfNodes()
	if err != nil {
		return nil, err
	}
	var appInfo monitor.ApplicationInfo
	for _, id := range appList {
		storage, err := newWithApplication(s.client, s.appName, id)
		if err != nil {
			return nil, err
		}
		appInfo.Merge(storage.ApplicationInfo())
	}
	return &appInfo, nil
}

// TaskInfo retuns information about the task
func (s *ClusterInfoReader) TaskInfo(name string) (*monitor.TaskInfo, error) {
	appList, err := s.ListOfNodes()
	if err != nil {
		return nil, err
	}
	var taskInfo monitor.TaskInfo
	for _, id := range appList {
		storage, err := newWithApplication(s.client, s.appName, id)
		if err != nil {
			return nil, err
		}
		info, err := storage.TaskInfo(name)
		if err != nil {
			return nil, err
		}
		taskInfo.Add(info)
	}
	return &taskInfo, nil
}

// TaskInfoByID retuns information about the particular task
func (s *ClusterInfoReader) TaskInfoByID(id string) (*monitor.TaskInfo, error) {
	appList, err := s.ListOfNodes()
	if err != nil {
		return nil, err
	}
	var taskInfo monitor.TaskInfo
	for _, id := range appList {
		storage, err := newWithApplication(s.client, s.appName, id)
		if err != nil {
			return nil, err
		}
		info, err := storage.TaskInfoByID(id)
		if err != nil {
			return nil, err
		}
		taskInfo.Add(info)
	}
	return &taskInfo, nil
}

// ListOfNodes returns list of registered nodes
func (s *ClusterInfoReader) ListOfNodes() ([]string, error) {
	s.mx.Lock()
	defer s.mx.Unlock()
	if len(s.appID) > 0 {
		return s.appID, nil
	}
	prefix := fmt.Sprintf("%s:app_", s.appName)
	resp := s.client.Keys(prefix + "*")
	if resp.Err() != nil {
		return nil, resp.Err()
	}
	list := make([]string, 0, len(resp.Val()))
	for _, val := range resp.Val() {
		list = append(list, strings.TrimPrefix(gocast.ToString(val), prefix))
	}
	return list, nil
}

// ResetCache of the nodes
func (s *ClusterInfoReader) ResetCache() {
	s.appID = s.appID[:0]
}
