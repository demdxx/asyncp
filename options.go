package asyncp

import (
	"context"
	"net"
	"os"

	"github.com/demdxx/asyncp/monitor"
)

type (
	// ContextWrapperFnk for prepare execution context
	ContextWrapperFnk func(ctx context.Context) context.Context
	// PanicHandlerFnk for any panic errors
	PanicHandlerFnk func(Task, Event, interface{})
	// ErrorHandlerFnk for any error response
	ErrorHandlerFnk func(Task, Event, error)
)

// Options of the mux server
type Options struct {
	MainExecContext context.Context
	PanicHandler    PanicHandlerFnk
	ErrorHandler    ErrorHandlerFnk
	ContextWrapper  ContextWrapperFnk
	ResponseFactory ResponseWriterFactory
	Cluster         ClusterExt
	EventAllocator  EventAllocator
}

func (opt *Options) _eventAllocator() EventAllocator {
	if opt.EventAllocator == nil {
		return newDefaultEventAllocator()
	}
	return opt.EventAllocator
}

// Option of the task configuration
type Option func(opt *Options)

// WithMainExecContext puts main execution context to the Mux option
func WithMainExecContext(ctx context.Context) Option {
	return func(opt *Options) {
		opt.MainExecContext = ctx
	}
}

// WithPanicHandler puts panic handler to the Mux option
func WithPanicHandler(h PanicHandlerFnk) Option {
	return func(opt *Options) {
		opt.PanicHandler = h
	}
}

// WithErrorHandler puts error handler to the Mux option
func WithErrorHandler(h ErrorHandlerFnk) Option {
	return func(opt *Options) {
		opt.ErrorHandler = h
	}
}

// WithContextWrapper puts context wrapper to the Mux option
func WithContextWrapper(w ContextWrapperFnk) Option {
	return func(opt *Options) {
		opt.ContextWrapper = w
	}
}

// WithStreamResponseMap set option with stream mapping converted to factory
func WithStreamResponseMap(streams ...interface{}) Option {
	return func(opt *Options) {
		opt.ResponseFactory = NewMultistreamResponseFactory(streams...)
	}
}

// WithResponseFactory set option with stream factory
func WithResponseFactory(responseFactory ResponseWriterFactory) Option {
	return func(opt *Options) {
		opt.ResponseFactory = responseFactory
	}
}

// WithStreamResponsePublisher set option with single stream publisher
func WithStreamResponsePublisher(publisher Publisher) Option {
	return func(opt *Options) {
		opt.ResponseFactory = NewStreamResponseFactory(publisher)
	}
}

// WithMonitor set option with monitoring storage
func WithMonitor(appName, host, hostname string, updater ...monitor.MetricUpdater) Option {
	return WithCluster(appName,
		ClusterWithHostinfo(host, hostname),
		ClusterWithStores(updater...),
	)
}

// WithMonitorDefaults set option with monitoring storage
func WithMonitorDefaults(appName string, updater ...monitor.MetricUpdater) Option {
	hostname, _ := os.Hostname()
	return WithMonitor(appName, localIP(), hostname, updater...)
}

// WithCluster set option of the cluster synchronizer from options
func WithCluster(appName string, options ...ClusterOption) Option {
	return WithClusterObject(NewCluster(appName, options...))
}

// WithClusterObject set option of the cluster synchronizer
func WithClusterObject(cluster ClusterExt) Option {
	return func(opt *Options) {
		opt.Cluster = cluster
	}
}

// WithEventAllocator set option with event allocator
func WithEventAllocator(eventAllocator EventAllocator) Option {
	return func(opt *Options) {
		opt.EventAllocator = eventAllocator
	}
}

func localIP() string {
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
