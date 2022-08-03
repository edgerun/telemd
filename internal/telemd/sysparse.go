package telemd

import (
	"bufio"
	"errors"
	"log"
	"os"
	"regexp"
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

type PsiMeasure struct {
	Avg10  float64
	Avg60  float64
	Avg300 float64
	Total  float64
}
type PsiResult struct {
	Some *PsiMeasure
	Full *PsiMeasure
}

func readPsiMeasure(line string) *PsiMeasure {
	//0: identifier (some/fulll), 1: avg10, 2: avg60, 3: avg300, 4: total
	fields := strings.Fields(line)

	parse := func(part string) float64 {
		kbstr := strings.Split(part, "=")
		value := kbstr[1]
		parsed, _ := strconv.ParseFloat(value, 64)
		return parsed
	}

	avg10 := parse(fields[1])
	avg60 := parse(fields[2])
	avg300 := parse(fields[3])
	total := parse(fields[4])

	return &PsiMeasure{
		Avg10:  avg10,
		Avg60:  avg60,
		Avg300: avg300,
		Total:  total,
	}
}

func parseIw(device string, attribute string, index int) (string, error) {
	args := "iw dev " + device + " link | awk -F '[ ]' '/" + attribute + ":/{print $" + strconv.Itoa(index) + "}'"
	result, err := execCommand(args)
	if err != nil {
		return result, err
	}
	return result, nil
}

func readPsiResult(resource string) (*PsiResult, error) {
	path := "/proc/pressure/" + resource

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var fullMeasure *PsiMeasure = nil

	scanner.Scan() // some should always exist
	someMeasure := readPsiMeasure(scanner.Text())

	// check if full is in the file
	if scanner.Scan() {
		fullMeasure = readPsiMeasure(scanner.Text())
	}

	return &PsiResult{Some: someMeasure, Full: fullMeasure}, nil
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

//
// Parses the net/dev file of a specific process, as follows:

// thomas@om ~ % cat /proc/114204/net/dev
// Inter-|   Receive                                                |  Transmit
//  face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
//   eth0:    6391      29    0    0    0     0          0         0        0       0    0    0    0     0       0          0
//     lo:       0       0    0    0    0     0          0         0        0       0    0    0    0     0       0          0
// returns the sum over all network devices (rx, tx)
func readTotalProcessNetworkStats(pid string, procMount string) (rx map[string]int64, tx map[string]int64, err error) {
	path := procMount + "/" + pid + "/net/dev"

	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Scan() // first header
	scanner.Scan() // second header
	rxValues := make(map[string]int64)
	txValues := make(map[string]int64)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		device := strings.TrimSuffix(fields[0], ":")
		irx, err := parseInt64(fields[1])
		if err != nil {
			return rx, tx, err
		}
		itx, err := parseInt64(fields[9])
		if err != nil {
			return rx, tx, err
		}

		rxValues[device] = irx
		txValues[device] = itx
	}

	return rxValues, txValues, scanner.Err()
}

func allPids(procFolder string) ([]string, error) {
	r, _ := regexp.Compile("^\\d*")
	dirs, err := listFilterDir(procFolder, func(info os.FileInfo) bool {
		return info.IsDir() && r.MatchString(info.Name())
	})

	if err != nil {
		log.Println("error fetching PIDs", err)
		return nil, err
	}
	return dirs, nil
}

func getContainerId(pid string, procMount string) (string, error) {
	// gets content of /proc/<pid>/cgroup
	// in cgroup v1 is list of multiple lines -> look for first that contains 'docker' substring
	// 11:freezer:/docker/dc65d1e5672961e7191260dec3dd532ad346719ea3ae23035e3b560867bd1183
	file, err := os.Open(procMount + "/" + pid + "/cgroup")
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "docker") {
			split := strings.Split(line, "/")[2]
			prefix := "docker-"
			suffix := ".scope"

			if strings.HasPrefix(split, prefix) {
				// cgroup v2
				return strings.TrimSuffix(strings.TrimPrefix(split, prefix), suffix), nil
			} else {
				// cgroup v1
				return split, nil
			}
		} else if strings.Contains(line, "kubepods") {
			// kubernetes
			// freezer:/kubepods/besteffort/podae778fdf-394c-4356-9625-ea50666783b1/2cc54a6877a50da0b6a2a5340dd1e8c5707a1d7d4b363e03b7cde76d2569f0c0
			return strings.Split(line, "/")[4], nil
		}
	}
	return "", errors.New("Did not find container for PID " + pid)
}

func containerProcessIds(procMount string) (map[string]string, error) {
	// get all PIDs from /proc
	pids, err := allPids(procMount)

	if err != nil {
		return nil, err
	}

	pidMap := make(map[string]string, 0)
	for _, pid := range pids {
		containerId, err := getContainerId(pid, procMount)
		if err == nil {
			if _, ok := pidMap[containerId]; !ok {
				pidMap[containerId] = pid
			}
		}
	}
	return pidMap, nil
}

// requires root
func containerProcessIdsUsingNsenter(procFolder string) (map[string]string, error) {
	// get all PIDs from /proc
	pids, err := allPids(procFolder)
	if err != nil {
		return nil, err
	}

	pidMap := make(map[string]string, 0)
	// execute for each PID 'nsenter -t $PID -u hostname'
	for _, pid := range pids {
		// check if result is equal to short containerId
		command, err := execCommand("nsenter -t " + pid + " -u hostname")
		if err == nil {
			pidMap[command] = pid
		}
	}
	return pidMap, nil
}

func containerProcessIdsUsingDockerCli() (map[string]string, error) {
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
