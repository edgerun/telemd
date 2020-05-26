package main

import (
	"github.com/edgerun/telemd/internal/env"
	"github.com/edgerun/telemd/internal/redis"
	"github.com/edgerun/telemd/internal/telem"
	"github.com/edgerun/telemd/internal/telemd"
	"log"
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

	reconnectingClient, err := redis.NewReconnectingClientFromUrl(cfg.Redis.URL, cfg.Redis.RetryBackoff)
	if err != nil {
		log.Fatal("could not create redis client: ", err)
	}
	daemon := telemd.NewDaemon(cfg)
	commandServer := telemd.NewRedisCommandServer(daemon, reconnectingClient.Client)
	telemetryReporter := telemd.NewRedisReporter(daemon, reconnectingClient.Client)

	go func() {
		for {
			state := <-reconnectingClient.ConnectionState
			switch state {
			case redis.Connected:
				go commandServer.Run()
				go telemetryReporter.Run()
				err := commandServer.UpdateNodeInfo()
				if err != nil {
					log.Fatal("error initializing node info", err)
				}
			case redis.Failed:
				daemon.PauseTickers()
				commandServer.Stop()
				go telemetryReporter.Stop()
				go reconnectingClient.Client.Ping()
			case redis.Recovered:
				daemon.UnpauseTickers()
				go commandServer.Run()
				go telemetryReporter.Run()
			default:
				return
			}
		}
	}()

	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		<-sigs
		log.Println("stopping daemon")
		commandServer.Stop()
		_ = commandServer.RemoveNodeInfo()
		telemetryReporter.Stop()
		daemon.Stop()
		close(reconnectingClient.ConnectionState)
	}()

	// initiate redis connection by sending a PING
	go reconnectingClient.Client.Ping()

	log.Println("running daemon")
	daemon.Run() // blocks until everything has shut down after daemon.Stop()
	log.Println("exiting")
}
