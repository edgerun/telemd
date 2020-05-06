package telem

import (
	"fmt"
	"os"
	"time"
)

const TopicSeparator = "/"

var NodeName, _ = os.Hostname()
var EmptyTelemetry = Telemetry{}

type Telemetry struct {
	Node  string
	Topic string
	Time  time.Time
	Value float64
}

type TelemetryChannel interface {
	Channel() chan Telemetry
	Put(telemetry Telemetry)
	Close()
}

type telemetryChannel struct {
	C chan Telemetry
}

func NewTelemetry(topic string, value float64) Telemetry {
	return Telemetry{
		Node:  NodeName,
		Topic: topic,
		Time:  time.Now(),
		Value: value,
	}
}

func NewTelemetryChannel() TelemetryChannel {
	c := make(chan Telemetry)
	return &telemetryChannel{
		C: c,
	}
}

func (m Telemetry) UnixTimeString() string {
	return fmt.Sprintf("%d.%d", m.Time.Unix(), m.Time.UnixNano()%m.Time.Unix())
}

func (m Telemetry) Print() {
	fmt.Printf("(%s, %s%s%s, %.4f)\n", m.Time, m.Node, TopicSeparator, m.Topic, m.Value)
}

func (t *telemetryChannel) Channel() chan Telemetry {
	return t.C
}

func (t *telemetryChannel) Put(telemetry Telemetry) {
	t.C <- telemetry
}

func (t *telemetryChannel) Close() {
	close(t.C)
}
