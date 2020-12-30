package redis

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis"
	"github.com/google/uuid"

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

	taskRegister bool
	taskLifetime time.Duration
}

// New storage connector
func New(opts ...Option) (*Storage, error) {
	var options Options
	for _, opt := range opts {
		opt(&options)
	}
	client, err := options.redisClient()
	return &Storage{
		client:       client,
		taskInfo:     map[string]*monitor.TaskInfo{},
		taskRegister: options.taskLifetime != 0,
		taskLifetime: options.taskLifetime,
	}, err
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
	return s.setJSON(s.mainKey(), appInfo, 0)
}

// DeregisterApplication info in the storage
func (s *Storage) DeregisterApplication() error {
	return s.client.Del(s.mainKey()).Err()
}

// ReceiveCount returns count of received messages
func (s *Storage) ReceiveCount() (uint64, error) {
	return s.client.Get(s.metricKey("receive")).Uint64()
}

// ReceiveEvent register and increments counters
func (s *Storage) ReceiveEvent(event monitor.EventType) error {
	if event.Err() != nil {
		return s.client.Incr(s.metricKey("receive_error")).Err()
	}
	pipe := s.client.TxPipeline()
	_ = pipe.Incr(s.metricKey("receive"))
	_, err := pipe.Exec()
	return err
}

// TaskInfo retuns information about the task
func (s *Storage) TaskInfo(name string) (*monitor.TaskInfo, error) {
	s.mx.RLock()
	taskInfo := s.taskInfo[name]
	s.mx.RUnlock()

	hasTaskInfo := false
	if taskInfo == nil {
		s.mx.Lock()
		defer s.mx.Unlock()
		hasTaskInfo = true
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
	if !hasTaskInfo {
		s.taskInfo[name] = taskInfo
	}
	return taskInfo, nil
}

// TaskInfoByID returns information about the particular task
func (s *Storage) TaskInfoByID(id string) (*monitor.TaskInfo, error) {
	taskInfo := &monitor.TaskInfo{}
	err := s.getJSON(s.metricKey(id), &taskInfo)
	if err != nil && err == redis.Nil {
		return nil, err
	}
	taskInfo.SuccessCount = taskInfo.TotalCount - taskInfo.ErrorCount
	return taskInfo, nil
}

// ExecuteTask commits the execution event status
func (s *Storage) ExecuteTask(event monitor.EventType, execTime time.Duration) error {
	taskInfo, errInfo := s.TaskInfo(event.Name())
	if errInfo != nil {
		return errInfo
	}
	pipe := s.client.TxPipeline()

	// Update the particular type with ID
	if s.taskRegister && event.ID() != uuid.Nil {
		eventID := event.ID().String()
		taskIDInfo, errIDInfo := s.TaskInfoByID(eventID)
		if errIDInfo != nil {
			return errIDInfo
		}
		taskIDInfo.Inc(event.Err(), execTime)
		taskIDInfo.AddTaskName(event.Name())
		if err := s.setJSON(s.metricKey(eventID), taskIDInfo, s.taskLifetime, pipe); err != nil {
			return err
		}
	}

	// Update general task information
	taskInfo.Inc(event.Err(), execTime)
	eventName := event.Name()
	pipe.Incr(s.metricKey(eventName + "_total"))
	if event.Err() != nil {
		pipe.Incr(s.metricKey(eventName + "_error"))
	}
	_ = pipe.MSet(
		s.metricKey(eventName+"_min"), int64(taskInfo.MinExecTime),
		s.metricKey(eventName+"_avg"), int64(taskInfo.AvgExecTime),
		s.metricKey(eventName+"_max"), int64(taskInfo.MaxExecTime),
	)
	_, execErr := pipe.Exec()
	return execErr
}

// FailoverTaskInfo retuns information about the failover task
func (s *Storage) FailoverTaskInfo(name string) (*monitor.TaskInfo, error) {
	return s.TaskInfo(failoverTaskName)
}

// ExecuteFailoverTask commits the execution event status
func (s *Storage) ExecuteFailoverTask(event monitor.EventType, execTime time.Duration) error {
	return s.ExecuteTask(
		monitor.WrapEventWithName(failoverTaskName, event),
		execTime)
}

func (s *Storage) setJSON(key string, value interface{}, expiration time.Duration, pipe ...redis.Pipeliner) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	var redisCmd redis.Cmdable = s.client
	if len(pipe) > 0 && pipe[0] != nil {
		redisCmd = pipe[0]
	}
	if expiration != 0 {
		return redisCmd.SetNX(key, data, expiration).Err()
	}
	return redisCmd.Set(key, data, expiration).Err()
}

func (s *Storage) getJSON(key string, target interface{}, pipe ...redis.Pipeliner) error {
	var (
		data string
		err  error
	)
	if len(pipe) > 0 && pipe[0] != nil {
		data, err = pipe[0].Get(key).Result()
	} else {
		data, err = s.client.Get(key).Result()
	}
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
