package telemd

import (
	"fmt"
	"github.com/edgerun/telemd/internal/telem"
	"github.com/go-redis/redis/v7"
	"log"
	"strings"
)

type RedisCommandServer struct {
	daemon  *Daemon
	client  *redis.Client
	stopped chan bool
}

func NewRedisCommandServer(daemon *Daemon, client *redis.Client) *RedisCommandServer {
	return &RedisCommandServer{
		daemon:  daemon,
		client:  client,
		stopped: make(chan bool),
	}
}

func (server *RedisCommandServer) Run() {
	topic := "telemcmd" + telem.TopicSeparator + telem.NodeName

	// TODO: retry loop if pubsub breaks
	pubsub := server.client.Subscribe(topic)
	channel := pubsub.Channel()

	for {
		select {
		case msg := <-channel:
			if msg == nil {
				// TODO: retry loop
				log.Println("pubsub empty")
				return
			}

			payload := msg.Payload
			log.Println("received command", payload)

			switch payload {
			case "pause":
				server.daemon.Send(Pause)
			case "unpause":
				server.daemon.Send(Unpause)
			case "info":
				err := server.UpdateNodeInfo()
				if err != nil {
					log.Println("error while updating node info", err)
				}
			default:
				log.Println("unhandled command", payload)
			}
		case stop := <-server.stopped:
			if stop {
				log.Println("closing pubsub")
				_ = pubsub.Close()
				return
			}
		}
	}
}

func (server *RedisCommandServer) UpdateNodeInfo() error {
	return WriteNodeInfo(server.client, server.daemon.cfg.NodeName, SysInfo())
}

func (server *RedisCommandServer) RemoveNodeInfo() error {
	return RemoveNodeInfo(server.client, server.daemon.cfg.NodeName)
}

func (server *RedisCommandServer) Stop() {
	server.stopped <- true
}

func WriteNodeInfo(client *redis.Client, nodeName string, info NodeInfo) error {
	key := "telemd.info:" + nodeName

	multi := client.TxPipeline()

	multi.HSet(key, "arch", info.Arch)
	multi.HSet(key, "boot", info.Boot)
	multi.HSet(key, "hostname", info.Hostname)
	multi.HSet(key, "ram", info.Ram)
	multi.HSet(key, "cpus", info.Cpus)
	multi.HSet(key, "disk", strings.Join(info.Disk, " "))
	multi.HSet(key, "net", strings.Join(info.Net, " "))

	_, err := multi.Exec()
	return err
}

func RemoveNodeInfo(client *redis.Client, nodeName string) error {
	return client.Del("telemd.info:" + nodeName).Err()
}

type RedisReporter struct {
	channel telem.TelemetryChannel
	client  *redis.Client
	stopped chan bool
}

func NewRedisReporter(daemon *Daemon, client *redis.Client) *RedisReporter {
	return &RedisReporter{
		channel: daemon.telemetry,
		client:  client,
		stopped: make(chan bool),
	}
}

// Run iterates over the configured TelemetryChannel and reports
// received Telemetry data through the configured redis client.
func (reporter *RedisReporter) Run() {
	for {
		select {
		case t := <-reporter.channel.Channel():
			receivers, err := report(reporter.client, t)

			if err != nil {
				// TODO proper error handling
				panic(err)
			}

			if receivers == 0 {
				// TODO: if there are no subscribers, we could pause this ticker for X seconds and try again
			}
		case stop := <-reporter.stopped:
			if stop {
				break
			}
		}
	}
}

func (reporter *RedisReporter) Stop() {
	reporter.stopped <- true
}

func report(client *redis.Client, t telem.Telemetry) (int64, error) {
	if t == telem.EmptyTelemetry {
		return 0, nil
	}

	channel := fmt.Sprintf("telem%s%s%s%s", telem.TopicSeparator, t.Node, telem.TopicSeparator, t.Topic)
	message := fmt.Sprintf("%s %f", t.UnixTimeString(), t.Value)
	cmd := client.Publish(channel, message)
	if cmd.Err() != nil {
		return 0, cmd.Err()
	}
	return cmd.Val(), nil
}
