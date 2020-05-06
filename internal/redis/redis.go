package redis

import goredis "github.com/go-redis/redis/v7"

func NewClient(options *goredis.Options) (*goredis.Client, error) {
	client := goredis.NewClient(options)
	options.MaxRetries = 100
	_, err := client.Ping().Result()
	return client, err
}

func NewClientFromUrl(url string) (*goredis.Client, error) {
	options, err := goredis.ParseURL(url)

	if err != nil {
		return nil, err
	}
	return NewClient(options)
}
