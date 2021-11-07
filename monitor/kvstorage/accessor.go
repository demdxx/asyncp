package kvstorage

import "time"

// KeyValueBasic provides basic data accessors
type KeyValueBasic interface {
	// Keys returns the list of keys with pattern "text"+*
	Keys(pattern string) ([]string, error)

	// Get value from key
	Get(key string) (interface{}, error)

	// MGet returns values from banch of keys in the same order
	MGet(keys ...string) ([]interface{}, error)

	// Incr value with key
	Incr(key string) (int64, error)

	// Set value for the key
	Set(key string, value interface{}, expiration ...time.Duration) error

	// MSet multiple set operation with {Key, Value, ...} input
	MSet(vals ...interface{}) error

	// Del keys from the storage
	Del(key ...string) error
}

// KeyValueTxAccessor defines accessor which will apply changes only after committing
type KeyValueTxAccessor interface {
	KeyValueBasic

	// Commit transaction changes
	Commit() error
}

// KeyValueAccessor defines the accessor with transaction
type KeyValueAccessor interface {
	KeyValueBasic

	// Begin new transaction
	Begin() (KeyValueTxAccessor, error)
}
