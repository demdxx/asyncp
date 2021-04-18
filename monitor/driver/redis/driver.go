package redis

import (
	"time"

	"github.com/demdxx/asyncp/monitor/kvstorage"
	"github.com/go-redis/redis"
)

// Accessor implementation of the redis key value accessor
type Accessor struct {
	conn redis.Cmdable
}

// New driver connection
func New(connectURL string) (*Accessor, error) {
	conn, err := connectRedis(connectURL)
	if err != nil {
		return nil, err
	}
	return &Accessor{conn: conn}, nil
}

// Keys returns the list of keys with pattern "text"+*
func (acc *Accessor) Keys(pattern string) ([]string, error) {
	return acc.conn.Keys(pattern).Result()
}

// Get value from key
func (acc *Accessor) Get(key string) (interface{}, error) {
	res, err := acc.conn.Get(key).Result()
	if err == redis.Nil {
		return res, kvstorage.ErrNil
	}
	return res, err
}

// MGet returns values from banch of keys in the same order
func (acc *Accessor) MGet(keys ...string) ([]interface{}, error) {
	return acc.conn.MGet(keys...).Result()
}

// Incr value with key
func (acc *Accessor) Incr(key string) (int64, error) {
	return acc.conn.Incr(key).Result()
}

// Set value for the key
func (acc *Accessor) Set(key string, value interface{}, expiration ...time.Duration) error {
	var exp time.Duration
	if len(expiration) > 0 {
		exp = expiration[0]
	}
	return acc.conn.Set(key, value, exp).Err()
}

// MSet multiple set operation with {Key, Value, ...} input
func (acc *Accessor) MSet(vals ...interface{}) error {
	return acc.conn.MSet(vals...).Err()
}

// Del keys from the storage
func (acc *Accessor) Del(key ...string) error {
	return acc.conn.Del(key...).Err()
}

// Begin new transaction
func (acc *Accessor) Begin() (kvstorage.KeyValueTxAccessor, error) {
	return &Accessor{conn: acc.conn.TxPipeline()}, nil
}

// Commit transaction changes
func (acc *Accessor) Commit() (err error) {
	if p, _ := acc.conn.(redis.Pipeliner); p != nil {
		_, err = p.Exec()
	}
	return err
}
