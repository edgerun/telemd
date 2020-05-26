package main

import (
	"github.com/edgerun/telemd/internal/env"
	"github.com/edgerun/telemd/internal/redis"
	"github.com/edgerun/telemd/internal/telem"
	"github.com/edgerun/telemd/internal/telemd"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
)

func loadConfig() *telemd.Config {
	cfg := telemd.NewDefaultConfig()
	cfg.LoadFromEnvironment(env.OsEnv) // load os env first to get potential telemd_nodename

	if _, err := os.Stat(telemd.DefaultConfigPath); err == nil {
		log.Println("reading config from", telemd.DefaultConfigPath)

		iniEnv, err := env.NewIniEnvironment(telemd.DefaultConfigPath)
		if err == nil {
			cfg.LoadFromEnvironment(iniEnv)
		}

		iniNodeEnv, err := env.NewIniSectionEnvironment(telemd.DefaultConfigPath, cfg.NodeName)
		if err == nil {
			cfg.LoadFromEnvironment(iniNodeEnv)
		}

		cfg.LoadFromEnvironment(env.OsEnv) // overwrite ini values with os env as per specified behavior
	}

	return cfg
}

func main() {
	cfg := loadConfig()

	telem.NodeName = cfg.NodeName
	hostname, _ := os.Hostname()
	log.Printf("starting telemd for node %s (hostname: %s)\n", telem.NodeName, hostname)

	daemon, err := telemd.NewDaemon(cfg)
	if err != nil {
		log.Fatal("error creating telemetry daemon: ", err)
	}

	redisClient, err := redis.NewClientFromUrl(cfg.Redis.URL)
	if nerr, ok := err.(*net.OpError); ok {
		// TODO: retry
		log.Fatal("could not connect to redis: ", nerr)
	}
	if err != nil {
		log.Fatal("error creating redis client ", err)
	}
	commandServer := telemd.NewRedisCommandServer(daemon, redisClient)
	err = commandServer.UpdateNodeInfo()
	if err != nil {
		log.Fatal("error initializing node info", err)
	}

	telemetryReporter := telemd.NewRedisReporter(daemon, redisClient)

	// exit handler
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		<-sigs
		log.Println("stopping daemon")
		commandServer.Stop()
		commandServer.RemoveNodeInfo()
		telemetryReporter.Stop()
		daemon.Stop()
	}()

	go commandServer.Run()
	go telemetryReporter.Run()

	log.Println("running daemon")
	daemon.Run() // blocks until everything has shut down after daemon.Stop()
	log.Println("exiting")
}
