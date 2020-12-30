package redis

import (
	"time"

	"github.com/go-redis/redis"
)

// Option type definition
type Option func(opt *Options)

// Options of the storage
type Options struct {
	redisURL     string
	client       redis.Cmdable
	taskLifetime time.Duration
}

func (opts *Options) redisClient() (redis.Cmdable, error) {
	if opts.client != nil {
		return opts.client, nil
	}
	var err error
	opts.client, err = connectRedis(opts.redisURL)
	return opts.client, err
}

// WithRedisURL returns option modifier with URL
func WithRedisURL(url string) Option {
	return func(opt *Options) {
		opt.redisURL = url
	}
}

// WithRedisClient returns option modifier with redis client
func WithRedisClient(client redis.Cmdable) Option {
	return func(opt *Options) {
		opt.client = client
	}
}

// WithTaskDetailInfo returns option modifier with detail task info
func WithTaskDetailInfo(lifetime time.Duration) Option {
	return func(opt *Options) {
		opt.taskLifetime = lifetime
	}
}
