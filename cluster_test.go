package asyncp

import (
	"context"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/demdxx/asyncp/v2/monitor"
	"github.com/stretchr/testify/assert"
)

type countingMetricUpdater struct {
	registered   atomic.Int32
	deregistered atomic.Int32
	received     atomic.Int32
	executed     atomic.Int32
	failovered   atomic.Int32
	lastEvent    monitor.EventType
}

func (m *countingMetricUpdater) RegisterApplication(*monitor.ApplicationInfo) error {
	m.registered.Add(1)
	return nil
}
func (m *countingMetricUpdater) DeregisterApplication() error {
	m.deregistered.Add(1)
	return nil
}
func (m *countingMetricUpdater) ReceiveEvent(event monitor.EventType) error {
	m.received.Add(1)
	m.lastEvent = event
	return nil
}
func (m *countingMetricUpdater) ExecuteTask(event monitor.EventType, _ time.Duration) error {
	m.executed.Add(1)
	m.lastEvent = event
	return nil
}
func (m *countingMetricUpdater) ExecuteFailoverTask(event monitor.EventType, _ time.Duration) error {
	m.failovered.Add(1)
	m.lastEvent = event
	return nil
}

type fakeClusterInfoReader struct {
	appInfo *monitor.ApplicationInfo
	err     error
	calls   atomic.Int32
}

func (r *fakeClusterInfoReader) ApplicationInfo() (*monitor.ApplicationInfo, error) {
	r.calls.Add(1)
	return r.appInfo, r.err
}
func (r *fakeClusterInfoReader) TaskInfo(string) (*monitor.TaskInfo, error)       { return nil, nil }
func (r *fakeClusterInfoReader) TaskInfoByID(string) (*monitor.TaskInfo, error)   { return nil, nil }
func (r *fakeClusterInfoReader) ListOfNodes() (map[string][]string, int, error) { return nil, 0, nil }

func TestClusterAllTasks(t *testing.T) {
	cluster := Cluster{
		appInfo: &monitor.ApplicationInfo{
			Tasks: map[string][]string{
				"a":   {"a.1"},
				"a.1": {"a.2"},
				"a.2": {},
			},
		},
	}
	expectedChains := map[string][]string{"a": {"a.1", "a.2"}}
	tasks := cluster.AllTasks()
	chains := cluster.AllTaskChains()
	if !assert.True(t, reflect.DeepEqual(cluster.appInfo.Tasks, tasks), "invalid all tasks format") {
		t.Error("Expected", cluster.appInfo.Tasks, "Actual", tasks)
	}
	if !assert.True(t, reflect.DeepEqual(expectedChains, chains), "invalid all chains format") {
		t.Error("Expected", expectedChains, "Actual", chains)
	}
}

func TestNewClusterPanicsWithoutReaderOrStore(t *testing.T) {
	assert.Panics(t, func() { NewCluster("test") })
}

func TestNewClusterDefaults(t *testing.T) {
	updater := &countingMetricUpdater{}
	cluster := NewCluster("test", ClusterWithStores(updater))
	assert.NotEmpty(t, cluster.hostname)
	assert.Equal(t, clusterDefaultSyncInterval, cluster.syncInterval)
}

func TestClusterNilReceiverIsSafe(t *testing.T) {
	var cluster *Cluster
	assert.NoError(t, cluster.RegisterApplication(context.Background(), NewTaskMux()))
	assert.NoError(t, cluster.UnregisterApplication())
	assert.NoError(t, cluster.ReceiveEvent(WithPayload(`test`, nil), nil))
	assert.NoError(t, cluster.ExecEvent(false, WithPayload(`test`, nil), 0, nil))
	assert.Nil(t, cluster.TargetEventsAfter(`test`))
	assert.Nil(t, cluster.AllTasks())
}

func TestClusterRegisterUnregisterApplication(t *testing.T) {
	updater := &countingMetricUpdater{}
	cluster := NewCluster("test", ClusterWithStores(updater))
	mux := NewTaskMux()
	_ = mux.Handle(`test`, testTask)

	assert.NoError(t, cluster.RegisterApplication(context.Background(), mux))
	assert.EqualValues(t, 1, updater.registered.Load())

	assert.NoError(t, cluster.UnregisterApplication())
	assert.EqualValues(t, 1, updater.deregistered.Load())
}

func TestClusterReceiveAndExecEvent(t *testing.T) {
	updater := &countingMetricUpdater{}
	cluster := NewCluster("test", ClusterWithStores(updater))

	event := WithPayload(`test`, `payload`)
	assert.NoError(t, cluster.ReceiveEvent(event, nil))
	assert.EqualValues(t, 1, updater.received.Load())

	assert.NoError(t, cluster.ExecEvent(false, event, time.Millisecond, nil))
	assert.EqualValues(t, 1, updater.executed.Load())

	assert.NoError(t, cluster.ExecEvent(true, event, time.Millisecond, assert.AnError))
	assert.EqualValues(t, 1, updater.failovered.Load())
	assert.ErrorIs(t, updater.lastEvent.Err(), assert.AnError)
}

func TestClusterReceiveEventWithNilEvent(t *testing.T) {
	updater := &countingMetricUpdater{}
	cluster := NewCluster("test", ClusterWithStores(updater))

	assert.NoError(t, cluster.ReceiveEvent(nil, assert.AnError))
	assert.EqualValues(t, 1, updater.received.Load())
	assert.ErrorIs(t, updater.lastEvent.Err(), assert.AnError)
}

func TestClusterSyncInfo(t *testing.T) {
	reader := &fakeClusterInfoReader{appInfo: &monitor.ApplicationInfo{
		Tasks: map[string][]string{"a": {"a.1"}},
	}}
	cluster := NewCluster("test", ClusterWithReader(reader))

	assert.NoError(t, cluster.SyncInfo())
	assert.ElementsMatch(t, []string{"a.1"}, cluster.TargetEventsAfter("a"))
	assert.Nil(t, cluster.TargetEventsAfter("missing"))
}

func TestClusterSyncInfoNoInfo(t *testing.T) {
	reader := &fakeClusterInfoReader{appInfo: nil}
	cluster := NewCluster("test", ClusterWithReader(reader))

	assert.ErrorIs(t, cluster.SyncInfo(), ErrNoSyncInformation)
}

func TestClusterSyncInfoReaderError(t *testing.T) {
	reader := &fakeClusterInfoReader{err: assert.AnError}
	cluster := NewCluster("test", ClusterWithReader(reader))

	assert.ErrorIs(t, cluster.SyncInfo(), assert.AnError)
}

func TestClusterSyncInfoWithoutReader(t *testing.T) {
	cluster := NewCluster("test", ClusterWithStores(&countingMetricUpdater{}))
	assert.NoError(t, cluster.SyncInfo())
}

func TestClusterStopSyncIdempotent(t *testing.T) {
	cluster := NewCluster("test", ClusterWithStores(&countingMetricUpdater{}))
	// StopSync must be a safe no-op when nothing was ever started.
	cluster.StopSync()
	cluster.StopSync()
}

func TestClusterRunSyncWithoutReaderReturnsImmediately(t *testing.T) {
	cluster := NewCluster("test", ClusterWithStores(&countingMetricUpdater{}))
	done := make(chan struct{})
	go func() {
		cluster.RunSync(context.Background())
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("RunSync should return immediately when infoReader is nil")
	}
}

func TestClusterRunSyncStopsOnContextDone(t *testing.T) {
	reader := &fakeClusterInfoReader{appInfo: &monitor.ApplicationInfo{}}
	cluster := NewCluster("test", ClusterWithReader(reader), ClusterWithSyncInterval(time.Hour))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		cluster.RunSync(ctx)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("RunSync did not stop after context cancellation")
	}
}

func TestClusterRunSyncTicks(t *testing.T) {
	reader := &fakeClusterInfoReader{appInfo: &monitor.ApplicationInfo{
		Tasks: map[string][]string{"a": {"b"}},
	}}
	cluster := NewCluster("test", ClusterWithReader(reader), ClusterWithSyncInterval(5*time.Millisecond))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		cluster.RunSync(ctx)
		close(done)
	}()

	assert.Eventually(t, func() bool {
		return reader.calls.Load() >= 1
	}, time.Second, 5*time.Millisecond)

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("RunSync did not stop after context cancellation")
	}
}
