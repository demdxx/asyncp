package asyncp

import (
	"time"

	"github.com/demdxx/asyncp/monitor"
)

// Monotor provides examination hook methods
type Monotor struct {
	appName  string
	host     string
	hostname string
	storages []monitor.MetricUpdater
}

// NewMonitor accessor
func NewMonitor(appName, host, hostname string, storage ...monitor.MetricUpdater) *Monotor {
	if host == "" {
		host = localIP()
	}
	return &Monotor{
		appName:  appName,
		host:     host,
		hostname: hostname,
		storages: storage,
	}
}

func (m *Monotor) register(mux *TaskMux) error {
	if m == nil {
		return nil
	}
	appInfo := &monitor.ApplicationInfo{
		Name:     m.appName,
		Host:     m.host,
		Hostname: m.hostname,
		Tasks:    mux.EventMap(),
	}
	for _, st := range m.storages {
		if err := st.RegisterApplication(appInfo); err != nil {
			return err
		}
	}
	return nil
}

func (m *Monotor) deregister() {
	if m == nil {
		return
	}
	for _, st := range m.storages {
		_ = st.DeregisterApplication()
	}
}

func (m *Monotor) receiveEvent(event Event, err error) {
	if m == nil {
		return
	}
	for _, st := range m.storages {
		_ = st.IncReceiveCount()
	}
}

func (m *Monotor) execEvent(failover bool, event Event, execTime time.Duration, err error) {
	if m == nil {
		return
	}
	for _, st := range m.storages {
		if failover {
			_ = st.IncFailoverTaskInfo(execTime, err)
		} else {
			_ = st.IncTaskInfo(event.Name(), execTime, err)
		}
	}
}
