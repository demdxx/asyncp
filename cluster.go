package asyncp

import (
	"context"
	"log"
	"os"
	"sync"
	"time"

	"go.uber.org/multierr"

	"github.com/demdxx/asyncp/monitor"
)

const (
	clusterDefaultSyncInterval = time.Second * 10
)

// ClusterOption provides option extractor for the cluster configuration
type ClusterOption = func(cluster *Cluster)

// ClusterWithHostinfo puts hostname information
func ClusterWithHostinfo(hostIP, hostname string) ClusterOption {
	return func(cluster *Cluster) {
		cluster.hostIP = hostIP
		cluster.hostname = hostname
	}
}

// ClusterWithSyncInterval between exec graph refresh
func ClusterWithSyncInterval(syncInterval time.Duration) ClusterOption {
	return func(cluster *Cluster) {
		cluster.syncInterval = syncInterval
	}
}

// ClusterWithReader of the culter info
func ClusterWithReader(infoReader monitor.ClusterInfoReader) ClusterOption {
	return func(cluster *Cluster) {
		cluster.infoReader = infoReader
	}
}

// ClusterWithStores between exec graph refresh
func ClusterWithStores(stores ...monitor.MetricUpdater) ClusterOption {
	return func(cluster *Cluster) {
		cluster.clusterStores = stores
	}
}

// ClusterExt extends functionality of mux
type ClusterExt interface {
	RegisterApplication(ctx context.Context, mux *TaskMux) error
	UnregisterApplication() error
	ReceiveEvent(event Event, err error) error
	ExecEvent(failover bool, event Event, execTime time.Duration, err error) error
	TargetEventsAfter(eventName string) []string
	AllTasks() map[string][]string
	AllTaskChains() map[string][]string
}

// Cluster provides synchronization of several processing pools
// and join all processing graphs in one cross-service execution map.
type Cluster struct {
	mx sync.RWMutex

	appName  string
	hostIP   string
	hostname string

	// interval of cluster infor updating
	syncInterval time.Duration
	tickInterval *time.Ticker

	infoReader    monitor.ClusterInfoReader
	clusterStores []monitor.MetricUpdater

	taskMap map[string][]string
	appInfo *monitor.ApplicationInfo

	mux *TaskMux
}

// NewCluster graph synchronization
func NewCluster(appName string, options ...ClusterOption) *Cluster {
	cluster := &Cluster{
		appName:      appName,
		syncInterval: clusterDefaultSyncInterval,
	}
	for _, opt := range options {
		opt(cluster)
	}
	if cluster.syncInterval <= 0 {
		cluster.syncInterval = clusterDefaultSyncInterval
	}
	if cluster.hostIP == "" {
		cluster.hostIP = localIP()
	}
	if cluster.hostname == "" {
		cluster.hostname, _ = os.Hostname()
	}
	if cluster.infoReader == nil && len(cluster.clusterStores) == 0 {
		panic("monitor.ClusterInfoReader or monitor.MetricUpdater is required for synchronisation")
	}
	return cluster
}

// RegisterApplication in iternal storages
func (cluster *Cluster) RegisterApplication(ctx context.Context, mux *TaskMux) (err error) {
	if cluster == nil {
		return nil
	}
	cluster.mux = mux
	appInfo := &monitor.ApplicationInfo{
		Name:     cluster.appName,
		Host:     cluster.hostIP,
		Hostname: cluster.hostname,
		Tasks:    mux.TaskMap(),
	}
	for _, up := range cluster.clusterStores {
		if regErr := up.RegisterApplication(appInfo); regErr != nil {
			err = multierr.Append(err, regErr)
		}
	}
	go cluster.RunSync(ctx)
	return err
}

// UnregisterApplication from cluster
func (cluster *Cluster) UnregisterApplication() (err error) {
	if cluster == nil {
		return nil
	}
	for _, st := range cluster.clusterStores {
		err = multierr.Append(err, st.DeregisterApplication())
	}
	cluster.StopSync()
	return err
}

// ReceiveEvent handler before it was executed
func (cluster *Cluster) ReceiveEvent(event Event, err error) error {
	if cluster == nil {
		return nil
	}
	targetEvent := monitor.EventType(event)
	if event == nil {
		targetEvent = monitor.NewErrorEvent(err)
	} else if err != nil {
		targetEvent = event.WithError(err)
	}
	var resErr error
	for _, st := range cluster.clusterStores {
		resErr = multierr.Append(resErr, st.ReceiveEvent(targetEvent))
	}
	return resErr
}

// ExecEvent handler after event has been executed
func (cluster *Cluster) ExecEvent(failover bool, event Event, execTime time.Duration, err error) error {
	if cluster == nil {
		return nil
	}
	targetEvent := monitor.EventType(event)
	if event == nil {
		targetEvent = monitor.NewErrorEvent(err)
	} else if err != nil {
		targetEvent = event.WithError(err)
	}
	var resErr error
	for _, up := range cluster.clusterStores {
		if failover {
			resErr = multierr.Append(resErr, up.ExecuteFailoverTask(targetEvent, execTime))
		} else {
			resErr = multierr.Append(resErr, up.ExecuteTask(targetEvent, execTime))
		}
	}
	return resErr
}

// TargetEventsAfter returns list of events to execute after the current event
func (cluster *Cluster) TargetEventsAfter(eventName string) []string {
	cluster.mx.RLock()
	defer cluster.mx.RUnlock()
	if cluster.taskMap == nil {
		return nil
	}
	return cluster.taskMap[eventName]
}

// AllTasks returns all events and links
func (cluster *Cluster) AllTasks() map[string][]string {
	if cluster.appInfo == nil {
		return map[string][]string{}
	}
	return cluster.appInfo.Tasks
}

// AllTaskChains returns all events task chains
func (cluster *Cluster) AllTaskChains() map[string][]string {
	var (
		tasks       = cluster.AllTasks()
		chains      = map[string][]string{}
		canContinue = true
	)
	for k, v := range tasks {
		chains[k] = append(make([]string, 0, len(v)), v...)
	}
mainLoop:
	for canContinue && len(chains) > 0 {
		for k, v := range chains {
			canContinue = false
			for _, key := range v {
				if len(chains[key]) > 0 {
					chains[k] = append(v, chains[key]...)
					canContinue = true
				}
				delete(chains, key)
			}
			if canContinue {
				continue mainLoop
			}
		}
	}
	return chains
}

// RunSync of cluster information
func (cluster *Cluster) RunSync(ctx context.Context) {
	if cluster.infoReader == nil {
		return
	}

	cluster.StopSync()
	defer cluster.StopSync()

	cluster.mx.Lock()
	cluster.tickInterval = time.NewTicker(cluster.syncInterval)
	tickChan := cluster.tickInterval.C
	cluster.mx.Unlock()
	for {
		select {
		case _, ok := <-tickChan:
			if !ok {
				return
			}
			if err := cluster.SyncInfo(); err != nil {
				log.Print(err)
			}
		case <-ctx.Done():
			return
		}
	}
}

// StopSync interval processing
func (cluster *Cluster) StopSync() {
	cluster.mx.Lock()
	defer cluster.mx.Unlock()
	if cluster.tickInterval != nil {
		cluster.tickInterval.Stop()
		cluster.tickInterval = nil
	}
}

// SyncInfo of the cluster
func (cluster *Cluster) SyncInfo() error {
	if cluster.infoReader == nil {
		return nil
	}
	appInfo, err := cluster.infoReader.ApplicationInfo()
	if err != nil {
		return err
	}
	cluster.appInfo = &monitor.ApplicationInfo{}
	cluster.appInfo.Merge(appInfo)
	if appInfo.Tasks != nil {
		cluster.mx.Lock()
		defer cluster.mx.Unlock()
		cluster.taskMap = map[string][]string{}
		for k, v := range cluster.appInfo.Tasks {
			cluster.taskMap[k] = append([]string{}, v...)
		}
	}
	return nil
}
