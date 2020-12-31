package redis

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/demdxx/asyncp/monitor"
	"github.com/demdxx/gocast"
	"github.com/go-redis/redis"
	"go.uber.org/multierr"
)

// ClusterInfoReader overrides group node accessing
type ClusterInfoReader struct {
	mx sync.Mutex

	appName []string
	client  redis.Cmdable

	storageList   []*Storage
	lastUpdated   time.Time
	cacheLifetime time.Duration
}

// NewClusterInfoReader interface implementation
func NewClusterInfoReader(client redis.Cmdable, appName ...string) *ClusterInfoReader {
	return &ClusterInfoReader{
		appName:       appName,
		client:        client,
		cacheLifetime: time.Minute,
	}
}

// NewClusterInfoReaderByURL from URL
func NewClusterInfoReaderByURL(connectURL string, appName ...string) (*ClusterInfoReader, error) {
	redisClient, err := connectRedis(connectURL)
	if err != nil {
		return nil, err
	}
	return NewClusterInfoReader(redisClient, appName...), nil
}

// ApplicationInfo returns application information
func (s *ClusterInfoReader) ApplicationInfo() (*monitor.ApplicationInfo, error) {
	storageList, err := s.ListStorages()
	if err != nil {
		return nil, err
	}
	var appInfo monitor.ApplicationInfo
	for _, storage := range storageList {
		appInfo.Merge(storage.ApplicationInfo())
	}
	return &appInfo, nil
}

// TaskInfo retuns information about the task
func (s *ClusterInfoReader) TaskInfo(name string) (*monitor.TaskInfo, error) {
	storageList, err := s.ListStorages()
	if err != nil {
		return nil, err
	}
	var taskInfo monitor.TaskInfo
	for _, storage := range storageList {
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
	storageList, err := s.ListStorages()
	if err != nil {
		return nil, err
	}
	var taskInfo monitor.TaskInfo
	for _, storage := range storageList {
		info, err := storage.TaskInfoByID(id)
		if err != nil {
			return nil, err
		}
		taskInfo.Add(info)
	}
	return &taskInfo, nil
}

// ListOfNodes returns list of registered nodes
func (s *ClusterInfoReader) ListOfNodes() (map[string][]string, int, error) {
	res := map[string][]string{}
	count := 0
	for _, appName := range s.appName {
		prefix := fmt.Sprintf("%s:app_", appName)
		resp := s.client.Keys(prefix + "*")
		if resp.Err() != nil {
			return nil, count, resp.Err()
		}
		list := make([]string, 0, len(resp.Val()))
		for _, val := range resp.Val() {
			list = append(list, strings.TrimPrefix(gocast.ToString(val), prefix))
		}
		res[appName] = list
	}
	return res, count, nil
}

// ListStorages returns the list of redis storages
func (s *ClusterInfoReader) ListStorages() ([]*Storage, error) {
	s.mx.Lock()
	defer s.mx.Unlock()
	if len(s.storageList) > 0 && time.Since(s.lastUpdated) <= s.cacheLifetime {
		return s.storageList, nil
	}
	appList, count, err := s.ListOfNodes()
	if err != nil {
		return nil, err
	}
	storageList := make([]*Storage, 0, count)
	for appName, hosts := range appList {
		for _, host := range hosts {
			storage, errStorage := newWithApplication(s.client, appName, host)
			if err != nil {
				err = multierr.Append(err, errStorage)
			} else if storage != nil {
				storageList = append(storageList, storage)
			}
		}
	}
	s.storageList = storageList
	s.lastUpdated = time.Now()
	return storageList, err
}

// ResetCache of the nodes
func (s *ClusterInfoReader) ResetCache() {
	s.storageList = s.storageList[:0]
}
