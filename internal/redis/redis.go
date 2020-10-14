package redis

import (
	goredis "github.com/go-redis/redis/v7"
	"math"
	"time"
)

type ConnectionState uint8

const (
	Connected ConnectionState = 1 // first connection
	Failed    ConnectionState = 2 // re-connect failed
	Recovered ConnectionState = 3 // successfully re-connected after failure
	Stopped   ConnectionState = 4
)

type ReconnectingClient struct {
	Client          *goredis.Client
	ConnectionState chan ConnectionState
	limiter         *limiter
}

func NewReconnectingClient(options *goredis.Options, retryBackoff time.Duration) *ReconnectingClient {
	connectionState := make(chan ConnectionState)
	client := goredis.NewClient(options)
	options.MaxRetries = math.MaxInt32
	limiter := newLimiter(retryBackoff, connectionState)
	options.Limiter = limiter
	return &ReconnectingClient{client, connectionState, limiter}
}

func (c *ReconnectingClient) Close() {
	c.ConnectionState <- Stopped
	c.limiter.close()
	close(c.ConnectionState)
}

func (c *ReconnectingClient) IsRetrying() bool {
	return c.limiter.connectionFailures > 0
}

func NewReconnectingClientFromUrl(url string, retryBackoff time.Duration) (*ReconnectingClient, error) {
	options, err := goredis.ParseURL(url)

	if err != nil {
		return nil, err
	}
	return NewReconnectingClient(options, retryBackoff), nil
}
