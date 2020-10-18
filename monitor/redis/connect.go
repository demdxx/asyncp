package redis

import (
	"net/url"
	"strings"

	"github.com/demdxx/gocast"
	"github.com/go-redis/redis"
)

func connectRedis(connectURL string) (redis.Cmdable, error) {
	var (
		urlHost, err = url.Parse(connectURL)
		password     string
	)
	if err != nil {
		return nil, err
	}
	if urlHost.User != nil {
		password, _ = urlHost.User.Password()
	}
	redisClient := redis.NewClient(&redis.Options{
		DB:           gocast.ToInt(strings.Trim(urlHost.Path, `/`)),
		Addr:         urlHost.Host,
		Password:     password,
		PoolSize:     gocast.ToInt(urlHost.Query().Get(`pool`)),
		MaxRetries:   gocast.ToInt(urlHost.Query().Get(`max_retries`)),
		MinIdleConns: gocast.ToInt(urlHost.Query().Get(`idle_cons`)),
	})
	return redisClient, nil
}
