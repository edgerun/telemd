package main

import (
	"git.dsg.tuwien.ac.at/mc2/go-telemetry/internal/env"
	"git.dsg.tuwien.ac.at/mc2/go-telemetry/internal/telem"
	"git.dsg.tuwien.ac.at/mc2/go-telemetry/internal/telemd"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg := telemd.NewDefaultConfig()
	cfg.LoadFromEnvironment(env.OsEnv)

	daemon, err := telemd.NewDaemon(cfg)
	if err != nil {
		log.Fatal("error creating telemetry daemon: ", err)
	}

	redis, err := telem.NewRedisClientFromUrl(cfg.Redis.URL)
	if nerr, ok := err.(*net.OpError); ok {
		// TODO: retry
		log.Fatal("could not connect to redis: ", nerr)
	}
	if err != nil {
		log.Fatal("error creating redis client ", err)
	}
	commandServer := telemd.NewRedisCommandServer(daemon, redis)
	telemetryReporter := telemd.NewRedisReporter(daemon, redis)

	// exit handler
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		<-sigs
		log.Println("stopping daemon")
		commandServer.Stop()
		telemetryReporter.Stop()
		daemon.Stop()
	}()

	go commandServer.Run()
	go telemetryReporter.Run()

	log.Println("running daemon")
	daemon.Run() // blocks until everything has shut down after daemon.Stop()
	log.Println("exiting")
}
