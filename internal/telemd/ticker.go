package telemd

import (
	"github.com/edgerun/go-telemd/internal/telem"
	"time"
)

type TelemetryTicker interface {
	Run()
	Stop()
	Pause()
	Unpause()
}

type telemetryTicker struct {
	instrument Instrument
	telemetryC telem.TelemetryChannel
	done       chan bool
	pause      chan bool
	duration   time.Duration
	ticker     *time.Ticker
}

func NewTelemetryTicker(instrument Instrument, channel telem.TelemetryChannel, duration time.Duration) TelemetryTicker {
	return &telemetryTicker{
		instrument: instrument,
		telemetryC: channel,
		done:       make(chan bool),
		pause:      make(chan bool),
		duration:   duration,
	}
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
