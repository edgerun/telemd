package telemd

import (
	"github.com/edgerun/go-telemd/internal/telem"
	"log"
	"runtime"
	"sync"
	"time"
)

type Daemon struct {
	cfg         *Config
	cmds        *commandChannel
	telemetry   telem.TelemetryChannel
	instruments map[string]Instrument

	tickers map[string]TelemetryTicker
}

func NewDaemon(cfg *Config) (*Daemon, error) {
	td := &Daemon{
		cfg:       cfg,
		telemetry: telem.NewTelemetryChannel(),
		cmds:      newCommandChannel(),
		tickers:   make(map[string]TelemetryTicker),
	}

	td.initInstruments(NewInstrumentFactory(runtime.GOARCH))
	td.initTickers()

	return td, nil
}

func (daemon *Daemon) initInstruments(factory InstrumentFactory) {
	cfg := daemon.cfg

	daemon.instruments = map[string]Instrument{
		"cpu":  factory.NewCpuUtilInstrument(),
		"freq": factory.NewCpuFrequencyInstrument(),
		"load": factory.NewLoadInstrument(),
		"net":  factory.NewNetworkDataRateInstrument(cfg.Instruments.Net.Devices),
		"disk": factory.NewDiskDataRateInstrument(cfg.Instruments.Disk.Devices),
	}
}

func (daemon *Daemon) initTickers() {
	for k, instrument := range daemon.instruments {
		ticker := NewTelemetryTicker(instrument, daemon.telemetry, daemon.cfg.Agent.Periods[k])
		daemon.tickers[k] = ticker
	}
}

func (daemon *Daemon) startTickers() *sync.WaitGroup {
	var wg sync.WaitGroup

	// start tickers and add to wait group
	for _, ticker := range daemon.tickers {
		go func(t TelemetryTicker) {
			wg.Add(1)
			t.Run()
			wg.Done()
		}(ticker)
	}

	return &wg
}

func (daemon *Daemon) Run() {
	var wg sync.WaitGroup
	wg.Add(2)

	// run command loop
	go func() {
		daemon.runCommandLoop()
		wg.Done()
	}()

	// run tickers
	go func() {
		daemon.startTickers().Wait()
		wg.Done()
	}()

	wg.Wait()
	time.Sleep(1 * time.Second) // TODO: properly wait for all tickers to exit
	log.Println("closing telemetry channel")
	daemon.telemetry.Close()
}

func (daemon *Daemon) Send(command Command) {
	daemon.cmds.channel <- command
}

func (daemon *Daemon) Stop() {
	// stop accepting Daemon channel
	daemon.cmds.stop <- true

	// stop tickers
	for k, ticker := range daemon.tickers {
		log.Println("stopping ticker " + k)
		ticker.Stop()
	}
}
