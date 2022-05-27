package telemd

import (
	"github.com/edgerun/telemd/internal/telem"
	"log"
	"runtime"
	"sync"
	"time"
)

type Daemon struct {
	cfg               *Config
	cmds              *commandChannel
	isPausedByCommand bool
	telemetry         telem.TelemetryChannel
	instruments       map[string]Instrument

	tickers map[string]TelemetryTicker
}

func NewDaemon(cfg *Config) *Daemon {
	td := &Daemon{
		cfg:       cfg,
		telemetry: telem.NewTelemetryChannel(),
		cmds:      newCommandChannel(),
		tickers:   make(map[string]TelemetryTicker),
	}

	td.initInstruments(NewInstrumentFactory(runtime.GOARCH))
	td.initTickers()

	return td
}

func (daemon *Daemon) initInstruments(factory InstrumentFactory) {
	cfg := daemon.cfg

	instruments := map[string]Instrument{
		"cpu":                    factory.NewCpuUtilInstrument(),
		"freq":                   factory.NewCpuFrequencyInstrument(),
		"load":                   factory.NewLoadInstrument(),
		"procs":                  factory.NewProcsInstrument(),
		"ram":                    factory.NewRamInstrument(),
		"net":                    factory.NewNetworkDataRateInstrument(cfg.Instruments.Net.Devices),
		"disk":                   factory.NewDiskDataRateInstrument(cfg.Instruments.Disk.Devices),
		"psi_cpu":                factory.NewPsiCpuInstrument(),
		"psi_memory":             factory.NewPsiMemoryInstrument(),
		"psi_io":                 factory.NewPsiIoInstrument(),
		"docker_cgrp_cpu":        factory.NewDockerCgroupCpuInstrument(),
		"docker_cgrp_blkio":      factory.NewDockerCgroupBlkioInstrument(),
		"docker_cgrp_net":        factory.NewDockerCgroupNetworkInstrument(),
		"docker_cgrp_memory":     factory.NewDockerCgroupMemoryInstrument(),
		"kubernetes_cgrp_cpu":    factory.NewKubernetesCgroupCpuInstrument(),
		"kubernetes_cgrp_blkio":  factory.NewKubernetesCgroupBlkioInstrument(),
		"kubernetes_cgrp_memory": factory.NewKubernetesCgroupMemoryInstrument(),
		"kubernetes_cgrp_net":    factory.NewKubernetesCgroupNetInstrument(),
	}

	activeNetDevice, err := findActiveNetDevice()
	if err == nil {
		wirelessPath := "/sys/class/net/" + activeNetDevice + "/wireless"
		if fileDirExists(wirelessPath) {
			instruments["tx_bitrate"] = factory.NewWifiTxBitrateInstrument(activeNetDevice)
			instruments["rx_bitrate"] = factory.NewWifiRxBitrateInstrument(activeNetDevice)
			instruments["signal"] = factory.NewWifiSignalInstrument(activeNetDevice)
		}
	}

	if cfg.Instruments.Disable != nil && (len(cfg.Instruments.Disable) > 0) {
		log.Println("disabling instruments", cfg.Instruments.Disable)
		for _, instr := range cfg.Instruments.Disable {
			delete(instruments, instr)
		}
		daemon.instruments = instruments
	} else if cfg.Instruments.Enable != nil && (len(cfg.Instruments.Enable) > 0) {
		log.Println("enabling instruments", cfg.Instruments.Enable)
		daemon.instruments = make(map[string]Instrument, len(cfg.Instruments.Enable))

		for _, key := range cfg.Instruments.Enable {
			if value, ok := instruments[key]; ok {
				daemon.instruments[key] = value
			}
		}
	} else {
		daemon.instruments = instruments
	}
}

func (daemon *Daemon) initTickers() {
	for k, instrument := range daemon.instruments {
		period, ok := daemon.cfg.Instruments.Periods[k]
		if !ok {
			log.Println("warning: no period assigned for instrument", k, "using 1")
			period = 1 * time.Second
		}
		ticker := NewTelemetryTicker(instrument, daemon.telemetry, period)
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
