package telem

import (
	"git.dsg.tuwien.ac.at/mc2/go-telemetry/internal/env"
	"log"
	"os"
	"time"
)

type ApplicationConfig struct {
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

func NewApplicationConfig() *ApplicationConfig {
	return &ApplicationConfig{}
}

func NewDefaultApplicationConfig() *ApplicationConfig {
	cfg := NewApplicationConfig()

	cfg.NodeName, _ = os.Hostname()

	cfg.Redis.URL = "redis://localhost"

	cfg.Instruments.Net.Devices = NetworkDevices()
	cfg.Instruments.Disk.Devices = BlockDevices()

	cfg.Agent.Periods = map[string]time.Duration{
		"cpu":  500 * time.Millisecond,
		"freq": 250 * time.Millisecond,
		"load": 5 * time.Second,
		"net":  500 * time.Millisecond,
		"disk": 500 * time.Millisecond,
	}

	return cfg
}

func (cfg *ApplicationConfig) LoadFromEnvironment(env env.Environment) {

	if name, ok := env.Lookup("telemc_node_name"); ok {
		cfg.NodeName = name
	}

	if url, ok := env.Lookup("telemc_redis_url"); ok {
		cfg.Redis.URL = url
	} else if host, ok := env.Lookup("telemc_redis_host"); ok {
		if port, ok := env.Lookup("telemc_redis_port"); ok {
			cfg.Redis.URL = "redis://" + host + ":" + port
		} else {
			cfg.Redis.URL = "redis://" + host
		}
	}

	if devices, ok, err := env.LookupFields("telemc_net_devices"); err == nil && ok {
		cfg.Instruments.Net.Devices = devices
	} else if err != nil {
		log.Fatal("Error reading telemc_net_devices", err)
	}
	if devices, ok, err := env.LookupFields("telemc_disk_devices"); err == nil && ok {
		cfg.Instruments.Disk.Devices = devices
	} else if err != nil {
		log.Fatal("Error reading telemc_disk_devices", err)
	}

}
