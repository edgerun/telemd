package telemd

import (
	"fmt"
	"github.com/edgerun/telemd/internal/env"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const DefaultConfigPath string = "/etc/telemd/config.ini"

type Config struct {
	NodeName string
	Redis    struct {
		URL          string
		RetryBackoff time.Duration
	}
	Instruments struct {
		Periods map[string]time.Duration
		Net     struct {
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
	cfg.Redis.RetryBackoff = 5 * time.Second

	cfg.Instruments.Net.Devices = networkDevices()
	cfg.Instruments.Disk.Devices = blockDevices()

	cfg.Instruments.Periods = map[string]time.Duration{
		"cpu":   500 * time.Millisecond,
		"freq":  500 * time.Millisecond,
		"procs": 500 * time.Millisecond,
		"ram":   1 * time.Second,
		"load":  5 * time.Second,
		"net":   500 * time.Millisecond,
		"disk":  500 * time.Millisecond,
	}

	return cfg
}

func (cfg *Config) LoadFromEnvironment(env env.Environment) {

	if name, ok := env.Lookup("telemd_nodename"); ok {
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
	if backoffString, ok := env.Lookup("telemd_redis_Retry_backoff"); ok {
		backoffDuration, err := time.ParseDuration(backoffString)
		if err != nil {
			cfg.Redis.RetryBackoff = backoffDuration
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

	for instrument := range cfg.Instruments.Periods {
		key := "telemd_period_" + instrument

		if duration, ok, err := env.LookupDuration(key); err == nil && ok {
			log.Println("setting duration of", instrument, "to", duration)
			cfg.Instruments.Periods[instrument] = duration
		} else if err != nil {
			log.Fatal("Error reading "+key, err)
		}
	}

}

func listFilterDir(dirname string, predicate func(info os.FileInfo) bool) []string {
	dir, err := ioutil.ReadDir(dirname)
	if err != nil {
		panic(err)
	}

	files := make([]string, 0)

	for _, f := range dir {
		if predicate(f) {
			files = append(files, f.Name())
		}
	}

	return files
}

func networkDevices() []string {
	return listFilterDir("/sys/class/net", func(info os.FileInfo) bool {
		return !info.IsDir() && info.Name() != "lo"
	})
}

func blockDevices() []string {
	return listFilterDir("/sys/block", func(info os.FileInfo) bool {
		return !info.IsDir() && !strings.HasPrefix(info.Name(), "loop")
	})
}

func netSpeed() string {
	activeNetDevice := findActiveNetDevice()
	wirelessPath := "/sys/class/net/" + activeNetDevice + "/wireless"
	if fileDirExists(wirelessPath) {
		return findWifiSpeed(activeNetDevice)
	}else {
		path := "/sys/class/net/" + activeNetDevice + "/speed"
		speed, err := readFirstLine(path)
		check(err)
		return speed;
	}
	return ""
}
func findActiveNetDevice() string {
	args := "route | awk 'NR==3{print $8}'"
	return execCommand(args)
}

func findWifiSpeed(device string) string {
	args := "iw dev "+device+" link | awk -F '[ ]' '/tx bitrate:/{print $3}'"
	speed := execCommand(args)
	value, _ := strconv.ParseFloat(speed,32)
	return fmt.Sprint(int(value))
}
func execCommand(args string) string {
	cmd := exec.Command("sh","-c", args)
	if output,err := cmd.Output(); err!= nil {
		log.Printf( "Error executing command: %s",err)
	}else{
		log.Printf( "wifi bitrate: %s",output)
		return strings.TrimSpace(string(output))
	}
	return ""
}
