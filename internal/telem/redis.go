package telem

import (
	"fmt"
	"github.com/go-redis/redis/v7"
)

func NewRedisClient(options *redis.Options) *redis.Client {
	client := redis.NewClient(options)
	_, err := client.Ping().Result()
	check(err)
	return client
}

func NewRedisClientFromUrl(url string) *redis.Client {
	options, err := redis.ParseURL(url)
	check(err)
	return NewRedisClient(options)
}

func report(client *redis.Client, m Telemetry) {
	if m == EmptyTelemetry {
		panic("Cannot report empty measurement")
	}

	channel := fmt.Sprintf("telemetry:%s:%s", m.Node, m.Metric)
	message := fmt.Sprintf("%s %f", m.UnixTimeString(), m.Value)
	cmd := client.Publish(channel, message)
	if cmd.Err() != nil {
		panic(cmd.Err())
	}
}

// RunRedisReporter iterates over the given TelemetryChannel and reports
// received Telemetry data through the given redis client.
func RunRedisReporter(client *redis.Client, channel TelemetryChannel) {
	for t := range channel.Channel() {
		report(client, t)
	}
}
