package kvstorage

import (
	"time"
)

// Option type definition
type Option func(opt *Options)

// Options of the storage
type Options struct {
	kvclient     KeyValueAccessor
	taskLifetime time.Duration
}

// WithKVClient returns option modifier with KV client
func WithKVClient(client KeyValueAccessor) Option {
	return func(opt *Options) {
		opt.kvclient = client
	}
}

// WithTaskDetailInfo returns option modifier with detail task info
func WithTaskDetailInfo(lifetime time.Duration) Option {
	return func(opt *Options) {
		opt.taskLifetime = lifetime
	}
}
