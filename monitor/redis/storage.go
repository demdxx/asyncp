package redis

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis"

	"github.com/demdxx/asyncp/monitor"
	"github.com/demdxx/gocast"
)

const failoverTaskName = "$failover"

// Storage monitor implementation for redis
type Storage struct {
	mx sync.RWMutex

	client redis.Cmdable

	appInfo  *monitor.ApplicationInfo
	taskInfo map[string]*monitor.TaskInfo
}

// New storage connector
func New(client redis.Cmdable) *Storage {
	return &Storage{
		client:   client,
		taskInfo: map[string]*monitor.TaskInfo{},
	}
}

func newWithApplication(client redis.Cmdable, name, host string) (*Storage, error) {
	storage := &Storage{
		client:   client,
		taskInfo: map[string]*monitor.TaskInfo{},
	}
	if _, err := storage.loadApplicationInfo(name, host); err != nil {
		return nil, err
	}
	return storage, nil
}

// NewByURL returns new storage from URL
func NewByURL(connectURL string) (*Storage, error) {
	redisClient, err := connectRedis(connectURL)
	if err != nil {
		return nil, err
	}
	return New(redisClient), nil
}

// ApplicationInfo returns application information
func (s *Storage) ApplicationInfo() *monitor.ApplicationInfo {
	return s.appInfo
}

func (s *Storage) loadApplicationInfo(name, host string) (*monitor.ApplicationInfo, error) {
	mainKey := fmt.Sprintf("%s:app_%s", name, host)
	if err := s.getJSON(mainKey, &s.appInfo); err != nil {
		return nil, err
	}
	return s.appInfo, nil
}

// RegisterApplication info in the storage
func (s *Storage) RegisterApplication(appInfo *monitor.ApplicationInfo) error {
	s.appInfo = appInfo
	return s.setJSON(s.mainKey(), appInfo)
}

// DeregisterApplication info in the storage
func (s *Storage) DeregisterApplication() error {
	return s.client.Del(s.mainKey()).Err()
}

// ReceiveCount returns count of received messages
func (s *Storage) ReceiveCount() (uint64, error) {
	return s.client.Get(s.metricKey("receive")).Uint64()
}

// IncReceiveCount increments received message
func (s *Storage) IncReceiveCount() error {
	return s.client.Incr(s.metricKey("receive")).Err()
}

// TaskInfo retuns information about the task
func (s *Storage) TaskInfo(name string) (*monitor.TaskInfo, error) {
	s.mx.RLock()
	taskInfo := s.taskInfo[name]
	s.mx.RUnlock()

	if taskInfo == nil {
		s.mx.Lock()
		defer s.mx.Unlock()
		dataResp := s.client.MGet(
			s.metricKey(name+"_total"),
			s.metricKey(name+"_error"),
			s.metricKey(name+"_min"),
			s.metricKey(name+"_avg"),
			s.metricKey(name+"_max"),
		)
		if err := dataResp.Err(); err != nil {
			return nil, err
		}
		vals := dataResp.Val()
		taskInfo = &monitor.TaskInfo{
			TotalCount:   gocast.ToUint64(vals[0]),
			ErrorCount:   gocast.ToUint64(vals[1]),
			SuccessCount: 0,
			MinExecTime:  time.Duration(gocast.ToInt64(vals[2])),
			AvgExecTime:  time.Duration(gocast.ToInt64(vals[3])),
			MaxExecTime:  time.Duration(gocast.ToInt64(vals[4])),
		}
		taskInfo.SuccessCount = taskInfo.TotalCount - taskInfo.ErrorCount
	}
	return taskInfo, nil
}

// IncTaskInfo increments particular task info
func (s *Storage) IncTaskInfo(name string, execTime time.Duration, err error) error {
	taskInfo, err := s.TaskInfo(name)
	if err != nil {
		return err
	}
	taskInfo.Inc(err, execTime)
	pipe := s.client.TxPipeline()
	pipe.Incr(s.metricKey(name + "_total"))
	if err != nil {
		pipe.Incr(s.metricKey(name + "_error"))
	}
	pipe.Set(s.metricKey(name+"_min"), int64(taskInfo.MinExecTime), 0)
	pipe.Set(s.metricKey(name+"_avg"), int64(taskInfo.AvgExecTime), 0)
	pipe.Set(s.metricKey(name+"_max"), int64(taskInfo.MaxExecTime), 0)
	_, execErr := pipe.Exec()
	return execErr
}

// FailoverTaskInfo retuns information about the failover task
func (s *Storage) FailoverTaskInfo(name string) (*monitor.TaskInfo, error) {
	return s.TaskInfo(failoverTaskName)
}

// IncFailoverTaskInfo increments failover task info
func (s *Storage) IncFailoverTaskInfo(execTime time.Duration, err error) error {
	return s.IncTaskInfo(failoverTaskName, execTime, err)
}

func (s *Storage) setJSON(key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return s.client.Set(key, data, 0).Err()
}

func (s *Storage) getJSON(key string, target interface{}) error {
	data, err := s.client.Get(key).Result()
	if err != nil {
		if err == redis.Nil {
			err = nil
		}
		return err
	}
	return json.Unmarshal([]byte(data), target)
}

func (s *Storage) mainKey() string {
	return fmt.Sprintf("%s:app_%s", s.appInfo.Name, s.appInfo.Host)
}

func (s *Storage) metricKey(key string) string {
	return fmt.Sprintf("%s:metric_%s_$_%s", s.appInfo.Name, s.appInfo.Host, key)
}
