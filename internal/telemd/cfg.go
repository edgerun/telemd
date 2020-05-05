package telemd

import (
	"git.dsg.tuwien.ac.at/mc2/go-telemetry/internal/env"
	"git.dsg.tuwien.ac.at/mc2/go-telemetry/internal/telem"
	"log"
	"os"
	"time"
)

type Config struct {
	NodeName string
	Redis    struct {
		URL string
	}
	Agent struct {
		Periods map[string]time.Duration
	}
	Instruments struct {
		Net struct {
			Devices []string
		}
		Disk struct {
			Devices []string
		}
	}
}

func NewConfig() *Config {
	return &Config{}
}

func NewDefaultConfig() *Config {
	cfg := NewConfig()

	cfg.NodeName, _ = os.Hostname()

	cfg.Redis.URL = "redis://localhost"

	cfg.Instruments.Net.Devices = telem.NetworkDevices()
	cfg.Instruments.Disk.Devices = telem.BlockDevices()

	cfg.Agent.Periods = map[string]time.Duration{
		"cpu":  500 * time.Millisecond,
		"freq": 250 * time.Millisecond,
		"load": 5 * time.Second,
		"net":  500 * time.Millisecond,
		"disk": 500 * time.Millisecond,
	}

	return cfg
}

func (cfg *Config) LoadFromEnvironment(env env.Environment) {

	if name, ok := env.Lookup("telemd_node_name"); ok {
		cfg.NodeName = name
	}

	if url, ok := env.Lookup("telemd_redis_url"); ok {
		cfg.Redis.URL = url
	} else if host, ok := env.Lookup("telemd_redis_host"); ok {
		if port, ok := env.Lookup("telemd_redis_port"); ok {
			cfg.Redis.URL = "redis://" + host + ":" + port
		} else {
			cfg.Redis.URL = "redis://" + host
		}
	}

	if devices, ok, err := env.LookupFields("telemd_net_devices"); err == nil && ok {
		cfg.Instruments.Net.Devices = devices
	} else if err != nil {
		log.Fatal("Error reading telemd_net_devices", err)
	}
	if devices, ok, err := env.LookupFields("telemd_disk_devices"); err == nil && ok {
		cfg.Instruments.Disk.Devices = devices
	} else if err != nil {
		log.Fatal("Error reading telemd_disk_devices", err)
	}

}
