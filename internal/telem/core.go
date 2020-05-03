package telem

import (
	"fmt"
	"os"
	"time"
)

var NodeName, _ = os.Hostname()
var EmptyTelemetry = Telemetry{}

type Telemetry struct {
	Metric string
	Node   string
	Time   time.Time
	Value  float64
}

type TelemetryChannel interface {
	Channel() chan Telemetry
	Put(telemetry Telemetry)
	Close()
}

type Instrument interface {
	// MeasureAndReport executes a measurement, creates appropriate Telemetry
	// instances, and put them into the given TelemetryChannel.
	MeasureAndReport(telemetry TelemetryChannel)
}

type TelemetryTicker interface {
	Run()
	Stop()
	Pause()
	Unpause()
}

type InstrumentFactory interface {
	NewCpuFrequencyInstrument() Instrument
	NewCpuUtilInstrument() Instrument
	NewLoadInstrument() Instrument
	NewNetworkDataRateInstrument(string) Instrument
}

type telemetryTicker struct {
	instrument Instrument
	telemetryC TelemetryChannel
	done       chan bool
	pause      chan bool
	duration   time.Duration
	ticker     *time.Ticker
}

type telemetryChannel struct {
	C chan Telemetry
}

func NewTelemetry(metric string, value float64) Telemetry {
	return Telemetry{
		Metric: metric,
		Node:   NodeName,
		Time:   time.Now(),
		Value:  value,
	}
}

func NewTelemetryChannel() TelemetryChannel {
	c := make(chan Telemetry)
	return &telemetryChannel{
		C: c,
	}
}

func NewTelemetryTicker(instrument Instrument, channel TelemetryChannel, duration time.Duration) TelemetryTicker {
	return &telemetryTicker{
		instrument: instrument,
		telemetryC: channel,
		done:       make(chan bool),
		pause:      make(chan bool),
		duration:   duration,
	}
}

func (m Telemetry) UnixTimeString() string {
	return fmt.Sprintf("%d.%d", m.Time.Unix(), m.Time.UnixNano()%m.Time.Unix())
}

func (m Telemetry) Print() {
	fmt.Printf("(%s, %s:%s, %.4f)\n", m.Time, m.Node, m.Metric, m.Value)
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

func (ticker *telemetryTicker) Run() {
	ticker.ticker = time.NewTicker(ticker.duration)

	for {
		select {
		case done := <-ticker.done:
			if done {
				// could wait for active tickers to return here, but what is the overhead?
				return
			}
		case pause := <-ticker.pause:
			if pause {
			Pausing:
				for {
					select {
					case done := <-ticker.done:
						if done {
							// we received done while pausing
							return
						}
					case pause := <-ticker.pause:
						if !pause {
							break Pausing
						}
					}
				}
			}
		case <-ticker.ticker.C:
			go ticker.instrument.MeasureAndReport(ticker.telemetryC)
		}
	}
}

func (ticker *telemetryTicker) Pause() {
	ticker.pause <- true
	ticker.ticker.Stop()
}

func (ticker *telemetryTicker) Unpause() {
	ticker.ticker = time.NewTicker(ticker.duration)
	ticker.pause <- false
}

func (ticker *telemetryTicker) Stop() {
	ticker.done <- true
	ticker.ticker.Stop()
}
