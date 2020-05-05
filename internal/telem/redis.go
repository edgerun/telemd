package telem

import (
	"github.com/go-redis/redis/v7"
)

func NewRedisClient(options *redis.Options) (*redis.Client, error) {
	client := redis.NewClient(options)
	options.MaxRetries = 100
	_, err := client.Ping().Result()
	return client, err
}

func NewRedisClientFromUrl(url string) (*redis.Client, error) {
	options, err := redis.ParseURL(url)

	if err != nil {
		return nil, err
	}
	return NewRedisClient(options)
}
