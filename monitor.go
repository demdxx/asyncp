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
	updaters []monitor.MetricUpdater
}

// NewMonitor accessor
func NewMonitor(appName, host, hostname string, updater ...monitor.MetricUpdater) *Monotor {
	if host == "" {
		host = localIP()
	}
	return &Monotor{
		appName:  appName,
		host:     host,
		hostname: hostname,
		updaters: updater,
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
	for _, up := range m.updaters {
		if err := up.RegisterApplication(appInfo); err != nil {
			return err
		}
	}
	return nil
}

func (m *Monotor) deregister() {
	if m == nil {
		return
	}
	for _, st := range m.updaters {
		_ = st.DeregisterApplication()
	}
}

func (m *Monotor) receiveEvent(event Event, err error) {
	if m == nil {
		return
	}
	var targetEvent monitor.EventType = event
	if event == nil {
		targetEvent = monitor.NewErrorEvent(err)
	} else {
		targetEvent = event.WithError(err)
	}
	for _, st := range m.updaters {
		_ = st.ReceiveEvent(targetEvent)
	}
}

func (m *Monotor) execEvent(failover bool, event Event, execTime time.Duration, err error) {
	if m == nil {
		return
	}
	var targetEvent monitor.EventType = event
	if event == nil {
		targetEvent = monitor.NewErrorEvent(err)
	} else {
		targetEvent = event.WithError(err)
	}
	for _, up := range m.updaters {
		if failover {
			_ = up.ExecuteFailoverTask(targetEvent, execTime)
		} else {
			_ = up.ExecuteTask(targetEvent, execTime)
		}
	}
}
