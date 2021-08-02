package asyncp

import (
	"context"
	"log"
	"net"
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

// Cluster provides synchronization of several processing pools
// and join all processing graphs in one cross-service execution map
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
		cluster.hostIP = getLocalIP()
	}
	if cluster.hostname == "" {
		cluster.hostname, _ = os.Hostname()
	}
	if cluster.infoReader == nil {
		panic("monitor.ClusterInfoReader is required for synchronisation")
	}
	return cluster
}

// RegisterApplication in iternal storages
func (cluster *Cluster) RegisterApplication(mux *TaskMux) (err error) {
	if cluster == nil {
		return nil
	}
	cluster.mux = mux
	appInfo := &monitor.ApplicationInfo{
		Name:     cluster.appName,
		Host:     cluster.hostIP,
		Hostname: cluster.hostname,
		Tasks:    mux.EventMap(),
	}
	for _, up := range cluster.clusterStores {
		if regErr := up.RegisterApplication(appInfo); regErr != nil {
			err = multierr.Append(err, regErr)
		}
	}
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

// RunSync of cluster information
func (cluster *Cluster) RunSync(ctx context.Context) {
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
	appInfo, err := cluster.infoReader.ApplicationInfo()
	if err != nil {
		return err
	}
	if appInfo.Tasks != nil {
		cluster.mx.Lock()
		defer cluster.mx.Unlock()
		cluster.taskMap = appInfo.Tasks
	}
	return nil
}

// getLocalIP returns the non loopback local IP of the host
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}
