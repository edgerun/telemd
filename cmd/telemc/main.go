package main

import (
	"git.dsg.tuwien.ac.at/mc2/go-telemc/internal/telem"
	"log"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"
)

func atExit(fn func()) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		fn()
		time.Sleep(5 * time.Second) // acts as a "await termination" timer
		os.Exit(0)
	}()
}

func main() {
	factory := telem.NewInstrumentFactory(runtime.GOARCH)

	// TODO: externalize into config
	instruments := map[string]telem.Instrument{
		"cpu":  factory.NewCpuUtilInstrument(),
		"freq": factory.NewCpuFrequencyInstrument(),
		"load": factory.NewLoadInstrument(),
		"net":  factory.NewNetworkDataRateInstrument("enp5s0"),
		"disk":  factory.NewDiskDataRateInstrument("sdc"),
	}

	periods := map[string]time.Duration{
		"cpu":  500 * time.Millisecond,
		"freq": 250 * time.Millisecond,
		"load": 5 * time.Second,
		"net":  500 * time.Millisecond,
		"disk": 500 * time.Millisecond,
	}

	// main channel for communicating telemetry data
	telemetryChannel := telem.NewTelemetryChannel()

	// create tickers and register close functions
	tickers := make(map[string]telem.TelemetryTicker)
	var wg sync.WaitGroup

	for k, instrument := range instruments {
		ticker := telem.NewTelemetryTicker(instrument, telemetryChannel, periods[k])
		tickers[k] = ticker

		// start ticker and add to wait group
		go func(t telem.TelemetryTicker) {
			wg.Add(1)
			t.Run()
			wg.Done()
		}(ticker)
	}

	// reporter
	client := telem.NewRedisClientFromUrl("redis://localhost")

	// cleanup
	atExit(func() {
		for k, t := range tickers {
			log.Println("stopping ticker " + k)
			t.Stop()
		}
		log.Println("waiting for tickers to stop")
		wg.Wait()
		log.Println("closing telemetry channel")
		telemetryChannel.Close() // breaks the reporter loop
	})

	log.Println("starting reporter ...")
	telem.RunRedisReporter(client, telemetryChannel) // blocks main thread
	log.Println("closing redis client")
	_ = client.Close()
	log.Println("main thread returning")
}
