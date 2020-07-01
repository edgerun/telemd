package telemd

import (
	"bufio"
	"errors"
	"os"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Reads the statistics from https://www.kernel.org/doc/Documentation/block/stat.txt
// and returns an array where the indices correspond to the following values:
//  0   read I/Os       requests      number of read I/Os processed
//  1   read merges     requests      number of read I/Os merged with in-queue I/O
//  2   read sectors    sectors       number of sectors read
//  3   read ticks      milliseconds  total wait time for read requests
//  4   write I/Os      requests      number of write I/Os processed
//  5   write merges    requests      number of write I/Os merged with in-queue I/O
//  6   write sectors   sectors       number of sectors written
//  7   write ticks     milliseconds  total wait time for write requests
//  8   in_flight       requests      number of I/Os currently in flight
//  9   io_ticks        milliseconds  total time this block device has been active
// 10   time_in_queue   milliseconds  total wait time for all requests
// 11   discard I/Os    requests      number of discard I/Os processed
// 12   discard merges  requests      number of discard I/Os merged with in-queue I/O
// 13   discard sectors sectors       number of sectors discarded
// 14   discard ticks   milliseconds  total wait time for discard requests
func readBlockDeviceStats(dev string) ([]int64, error) {
	path := "/sys/block/" + dev + "/stat"

	line, err := readFirstLine(path)
	if err != nil {
		return nil, err
	}

	values, err := parseInt64Array(strings.Fields(line))
	if err != nil {
		return values, err
	}
	return values, nil
}

// readCpuUtil returns an array of the following values from /proc/stat
// user, nice, system, idle, iowait, irq, softirq
func readCpuUtil() []float64 {
	line, err := readFirstLine("/proc/stat")
	check(err)
	line = strings.Trim(line, " ")
	parts := strings.Split(line, " ")

	var values []float64
	for _, v := range parts[2:] { // first two parts are 'cpu' and a whitespace
		val, err := strconv.ParseFloat(v, 64)
		check(err)
		values = append(values, val)
	}

	return values
}

func readMeminfo() map[string]string {
	vals := make(map[string]string)

	parser := func(line string) bool {
		parts := strings.Split(line, ":")

		if len(parts) != 2 {
			return true
		}

		k, v := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		vals[k] = v

		return true
	}

	check(visitLines("/proc/meminfo", parser))

	return vals
}

func readMemTotal() (int64, error) {
	val, ok := readMeminfo()["MemTotal"]
	if !ok {
		return 0, errors.New("MemTotal not found")
	}
	return parseMeminfoString(val)
}

// Parses the given size string from /proc/meminfo and returns the value in kB.
func parseMeminfoString(sizeString string) (int64, error) {
	// we're assuming that /proc/meminfo always returns kb
	kbstr := strings.Split(sizeString, " ")
	value := kbstr[0]
	return strconv.ParseInt(value, 10, 64)
}

func readUptime() (float64, error) {
	line, err := readFirstLine("/proc/uptime")
	if err != nil {
		return 0, err
	}

	parts := strings.Split(line, " ")
	if len(parts) != 2 {
		return 0, errors.New("Unexpected number of fields in /proc/uptime: " + line)
	}

	return strconv.ParseFloat(parts[0], 64)
}

func bootTime() (int64, error) {
	uptime, err := readUptime()
	if err != nil {
		return 0, err
	}
	return time.Now().Unix() - int64(uptime), nil
}

func arm64Gpu() ([]string, error) {
	// this only works on jetson devices!
	// other way to list all gpu devices, though without knowing the jetson device:
	// all gpu devices are mounted in /sys/devices, i.e. /sys/devices/gpu.0 <- works also in container (L4T base image)

	// https://forums.developer.nvidia.com/t/how-to-identify-nano/72160
	id, err := readFirstLine("/sys/module/tegra_fuse/parameters/tegra_chip_id")
	if err != nil {
		return []string{}, err
	}

	if id == "64" {
		return []string{"0-Jetson TK1"}, nil
	} else if id == "33" {
		// according to the blog post above, jetson nano has the same id as tx1
		// problem: /proc/device-tree/model not available in container
		return []string{"0-Jetson TX1"}, nil
	} else if id == "24" {
		return []string{"0-Jetson TX2"}, nil
	} else {
		return []string{}, errors.New(fmt.Sprintf("unsupported tegra chip: %s", id))
	}
}


//
// Parses the net/dev file of a specific process, as follows:

// thomas@om ~ % cat /proc/114204/net/dev
// Inter-|   Receive                                                |  Transmit
//  face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
//   eth0:    6391      29    0    0    0     0          0         0        0       0    0    0    0     0       0          0
//     lo:       0       0    0    0    0     0          0         0        0       0    0    0    0     0       0          0
// returns the sum over all network devices (rx, tx)
func readTotalProcessNetworkStats(pid string) (rx int64, tx int64, err error) {
	path := "/proc/" + pid + "/net/dev"

	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Scan() // first header
	scanner.Scan() // second header

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())

		irx, err := parseInt64(fields[1])
		if err != nil {
			return rx, tx, err
		}
		itx, err := parseInt64(fields[9])
		if err != nil {
			return rx, tx, err
		}

		rx += irx
		tx += itx
	}

	return rx, tx, scanner.Err()
}

func containerProcessIds() (map[string]string, error) {
	command, err := execCommand("docker ps -q | xargs docker inspect --format '{{ .Id }} {{ .State.Pid }}'")

	if err != nil {
		return nil, err
	}

	lines := strings.Split(command, "\n")
	pidmap := make(map[string]string, len(lines))
	for _, line := range lines {
		fields := strings.Split(line, " ")
		pidmap[fields[0]] = fields[1]
	}

	return pidmap, err
}
