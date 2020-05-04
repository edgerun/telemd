package main

import (
	"git.dsg.tuwien.ac.at/mc2/go-telemc/internal/env"
	"git.dsg.tuwien.ac.at/mc2/go-telemc/internal/telem"
	"github.com/go-redis/redis/v7"
	"log"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

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

func commandLoop(pubsub *redis.PubSub, tickers map[string]telem.TelemetryTicker) {
	for {
		msg, err := pubsub.ReceiveMessage()
		if err != nil {
			break
		}

		switch msg.Payload {
		case "pause":
			log.Printf("pausing %d tickers\n", len(tickers))
			for _, ticker := range tickers {
				ticker.Pause()
			}
		case "unpause":
			log.Printf("unpausing %d tickers\n", len(tickers))
			for _, ticker := range tickers {
				ticker.Unpause()
			}
		default:
			log.Println("unhandled command", msg.Payload)
		}
	}
}

func main() {
	cfg := telem.NewDefaultApplicationConfig()
	cfg.LoadFromEnvironment(env.OsEnv)

	factory := telem.NewInstrumentFactory(runtime.GOARCH)

	instruments := map[string]telem.Instrument{
		"cpu":  factory.NewCpuUtilInstrument(),
		"freq": factory.NewCpuFrequencyInstrument(),
		"load": factory.NewLoadInstrument(),
		"net":  factory.NewNetworkDataRateInstrument(cfg.Instruments.Net.Devices),
		"disk": factory.NewDiskDataRateInstrument(cfg.Instruments.Disk.Devices),
	}

	// TODO: externalize into config
	periods := 	cfg.Agent.Periods
	telem.NodeName = cfg.NodeName

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

	// reporter/command loop
	client := telem.NewRedisClientFromUrl(cfg.Redis.URL)

	pubsub := client.Subscribe("telemcmd" + telem.TopicSeparator + telem.NodeName)
	go commandLoop(pubsub, tickers)

	// cleanup
	atExit(func() {
		_ = pubsub.Close()

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
