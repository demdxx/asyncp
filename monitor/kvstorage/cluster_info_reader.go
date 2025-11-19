package kvstorage

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/demdxx/asyncp/v2/monitor"
	"github.com/demdxx/gocast/v2"
	"go.uber.org/multierr"
)

// ClusterInfoReader overrides group node accessing
type ClusterInfoReader struct {
	mx sync.RWMutex

	appName  []string
	kvclient KeyValueAccessor

	storageList   []*Storage
	lastUpdated   time.Time
	cacheLifetime time.Duration
}

// NewClusterInfoReader interface implementation
func NewClusterInfoReader(kvclient KeyValueAccessor, appName ...string) *ClusterInfoReader {
	return &ClusterInfoReader{
		appName:       appName,
		kvclient:      kvclient,
		cacheLifetime: time.Minute,
	}
}

func (s *ClusterInfoReader) String() string {
	return "Cluster: " + strings.Join(s.appName, ",")
}

// ApplicationInfo returns application information
func (s *ClusterInfoReader) ApplicationInfo() (*monitor.ApplicationInfo, error) {
	storageList, err := s.ListStorages()
	if err != nil || len(storageList) == 0 {
		return nil, err
	}
	s.mx.RLock()
	defer s.mx.RUnlock()
	var appInfo monitor.ApplicationInfo
	for _, storage := range storageList {
		if storageAppInfo := storage.ApplicationInfo(); storageAppInfo != nil {
			appInfo.Merge(storageAppInfo)
		}
	}
	return &appInfo, nil
}

// TaskInfo returns information about the task
func (s *ClusterInfoReader) TaskInfo(name string) (*monitor.TaskInfo, error) {
	storageList, err := s.ListStorages()
	if err != nil {
		return nil, err
	}
	s.mx.RLock()
	defer s.mx.RUnlock()
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

// TaskInfoByID returns information about the particular task
func (s *ClusterInfoReader) TaskInfoByID(id string) (*monitor.TaskInfo, error) {
	storageList, err := s.ListStorages()
	if err != nil {
		return nil, err
	}
	s.mx.RLock()
	defer s.mx.RUnlock()
	taskInfo := monitor.TaskInfo{ID: id}
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
		keys, err := s.kvclient.Keys(prefix + "*")
		if err != nil {
			return nil, count, err
		}
		list := make([]string, 0, len(keys))
		for _, val := range keys {
			list = append(list, strings.TrimPrefix(gocast.Str(val), prefix))
		}
		res[appName] = list
	}
	return res, count, nil
}

// ListStorages returns the list of redis storages
func (s *ClusterInfoReader) ListStorages() ([]*Storage, error) {
	if s == nil {
		return nil, nil
	}
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
			storage, errStorage := newWithApplication(s.kvclient, appName, host)
			if errStorage != nil {
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
	s.mx.Lock()
	defer s.mx.Unlock()
	s.storageList = s.storageList[:0]
}
