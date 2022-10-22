package kvstorage

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/demdxx/asyncp/v2/libs/errors"
	"github.com/demdxx/asyncp/v2/monitor"
	"github.com/demdxx/gocast"
)

const failoverTaskName = "$failover"

// ErrNil in case of empty response
var ErrNil = errors.ErrNil

// Storage monitor implementation for redis
type Storage struct {
	mx sync.RWMutex

	client KeyValueAccessor

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
	return &Storage{
		client:       options.kvclient,
		taskInfo:     map[string]*monitor.TaskInfo{},
		taskRegister: options.taskLifetime != 0,
		taskLifetime: options.taskLifetime,
	}, nil
}

func newWithApplication(client KeyValueAccessor, name, host string) (*Storage, error) {
	storage := &Storage{
		client:   client,
		taskInfo: map[string]*monitor.TaskInfo{},
		appInfo:  &monitor.ApplicationInfo{},
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
	return s.client.Del(s.mainKey())
}

// ReceiveCount returns count of received messages
func (s *Storage) ReceiveCount() (uint64, error) {
	val, err := s.client.Get(s.metricKey("receive"))
	if err != nil {
		return 0, err
	}
	return gocast.ToUint64(val), nil
}

// ReceiveEvent register and increments counters
func (s *Storage) ReceiveEvent(event monitor.EventType) (err error) {
	if event.Err() != nil {
		_, err = s.client.Incr(s.metricKey("receive_error"))
	} else {
		_, err = s.client.Incr(s.metricKey("receive"))
	}
	return err
}

// TaskInfo returns information about the task
func (s *Storage) TaskInfo(name string) (*monitor.TaskInfo, error) {
	s.mx.RLock()
	taskInfo := s.taskInfo[name]
	s.mx.RUnlock()

	if taskInfo == nil {
		s.mx.Lock()
		defer s.mx.Unlock()
		vals, err := s.client.MGet(
			s.metricKey(name+"_total"),
			s.metricKey(name+"_error"),
			s.metricKey(name+"_skip"),
			s.metricKey(name+"_min"),
			s.metricKey(name+"_avg"),
			s.metricKey(name+"_max"),
		)
		if err != nil {
			return nil, err
		}
		taskInfo = &monitor.TaskInfo{
			TotalCount:   gocast.ToUint64(vals[0]),
			ErrorCount:   gocast.ToUint64(vals[1]),
			SkipCount:    gocast.ToUint64(vals[2]),
			SuccessCount: gocast.ToUint64(vals[0]) - gocast.ToUint64(vals[1]) - gocast.ToUint64(vals[2]),
			MinExecTime:  time.Duration(gocast.ToInt64(vals[3])),
			AvgExecTime:  time.Duration(gocast.ToInt64(vals[4])),
			MaxExecTime:  time.Duration(gocast.ToInt64(vals[5])),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		s.taskInfo[name] = taskInfo
	}
	return taskInfo, nil
}

// TaskInfoByID returns information about the particular task
func (s *Storage) TaskInfoByID(id string) (*monitor.TaskInfo, error) {
	taskInfo := &monitor.TaskInfo{}
	err := s.getJSON(s.metricKey(id), &taskInfo)
	if err != nil {
		return nil, err
	}
	taskInfo.SuccessCount = taskInfo.TotalCount - taskInfo.ErrorCount - taskInfo.SkipCount
	return taskInfo, nil
}

// ExecuteTask commits the execution event status
func (s *Storage) ExecuteTask(event monitor.EventType, execTime time.Duration) error {
	taskInfo, errInfo := s.TaskInfo(event.Name())
	if errInfo != nil {
		return errInfo
	}
	tx, err := s.client.Begin()
	if err != nil {
		return err
	}

	// Update the particular type with ID
	if s.taskRegister && event.ID() != uuid.Nil {
		eventID := event.ID().String()
		taskIDInfo, errIDInfo := s.TaskInfoByID(eventID)
		if errIDInfo != nil {
			return errIDInfo
		}
		taskIDInfo.Inc(event.Err(), execTime)
		taskIDInfo.AddTaskName(event.Name())
		if err := s.setJSON(s.metricKey(eventID), taskIDInfo, s.taskLifetime, tx); err != nil {
			return err
		}
	}

	// Update general task information
	taskInfo.Inc(event.Err(), execTime)
	eventName := event.Name()
	_, _ = tx.Incr(s.metricKey(eventName + "_total"))
	if event.Err() != nil {
		if errors.Is(event.Err(), errors.ErrSkipEvent) {
			_, _ = tx.Incr(s.metricKey(eventName + "_skip"))
		} else {
			_, _ = tx.Incr(s.metricKey(eventName + "_error"))
		}
	}
	_ = tx.MSet(
		s.metricKey(eventName+"_min"), int64(taskInfo.MinExecTime),
		s.metricKey(eventName+"_avg"), int64(taskInfo.AvgExecTime),
		s.metricKey(eventName+"_max"), int64(taskInfo.MaxExecTime),
	)
	return tx.Commit()
}

// FailoverTaskInfo returns information about the failover task
func (s *Storage) FailoverTaskInfo(name string) (*monitor.TaskInfo, error) {
	return s.TaskInfo(failoverTaskName)
}

// ExecuteFailoverTask commits the execution event status
func (s *Storage) ExecuteFailoverTask(event monitor.EventType, execTime time.Duration) error {
	return s.ExecuteTask(
		monitor.WrapEventWithName(failoverTaskName, event),
		execTime)
}

func (s *Storage) setJSON(key string, value any, expiration time.Duration, tx ...KeyValueBasic) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	kvacc := s.client.(KeyValueBasic)
	if len(tx) > 0 && tx[0] != nil {
		kvacc = tx[0]
	}
	return kvacc.Set(key, string(data), expiration)
}

func (s *Storage) getJSON(key string, target any, tx ...KeyValueBasic) error {
	var (
		err   error
		data  any
		kvacc = s.client.(KeyValueBasic)
	)
	if len(tx) > 0 && tx[0] != nil {
		kvacc = tx[0]
	}
	if data, err = kvacc.Get(key); err != nil {
		if err == ErrNil {
			err = nil
		}
		return err
	}
	return json.Unmarshal([]byte(gocast.ToString(data)), target)
}

func (s *Storage) mainKey() string {
	return fmt.Sprintf("%s:app_%s", s.appInfo.Name, s.appInfo.Host)
}

func (s *Storage) metricKey(key string) string {
	return fmt.Sprintf("%s:metric_%s_$_%s", s.appInfo.Name, s.appInfo.Host, key)
}
