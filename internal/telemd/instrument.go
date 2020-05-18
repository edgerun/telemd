package telemd

import (
	"bufio"
	"github.com/edgerun/go-telemd/internal/telem"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Instrument interface {
	// MeasureAndReport executes a measurement, creates appropriate Telemetry
	// instances, and put them into the given TelemetryChannel.
	MeasureAndReport(telemetry telem.TelemetryChannel)
}

type InstrumentFactory interface {
	NewCpuFrequencyInstrument() Instrument
	NewCpuUtilInstrument() Instrument
	NewLoadInstrument() Instrument
	NewNetworkDataRateInstrument([]string) Instrument
	NewDiskDataRateInstrument([]string) Instrument
}

type CpuInfoFrequencyInstrument struct{}
type CpuScalingFrequencyInstrument struct{}
type CpuUtilInstrument struct{}
type LoadInstrument struct{}
type RamInstrument struct{}
type NetworkDataRateInstrument struct {
	Devices []string
}
type DiskDataRateInstrument struct {
	Devices []string
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

func (CpuUtilInstrument) MeasureAndReport(channel telem.TelemetryChannel) {
	then := readCpuUtil()
	time.Sleep(500 * time.Millisecond)
	now := readCpuUtil()

	val := (now[0] - then[0] + now[2] - then[2]) * 100. / (now[0] - then[0] + now[2] - then[2] + now[3] - then[3])
	channel.Put(telem.NewTelemetry("cpu", val))
}

func (CpuInfoFrequencyInstrument) MeasureAndReport(channel telem.TelemetryChannel) {
	file, err := os.Open("/proc/cpuinfo")
	check(err)
	defer func() {
		err = file.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	scanner := bufio.NewScanner(file)

	var sum float64

	for scanner.Scan() {
		text := scanner.Text()

		if strings.HasPrefix(text, "cpu MHz") {
			split := strings.Split(text, ":")
			strval := strings.Trim(split[1], " ")
			if val, err := strconv.ParseFloat(strval, 64); err == nil {
				sum += val
			} else {
				log.Println("could not parse value: '", strval, "' to float:", err)
			}
		}

		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}
	}

	channel.Put(telem.NewTelemetry("freq", sum))
}

var cpuScalingFiles, _ = filepath.Glob("/sys/devices/system/cpu/cpu[0-9]*/cpufreq/scaling_cur_freq")

func (c CpuScalingFrequencyInstrument) MeasureAndReport(channel telem.TelemetryChannel) {
	var sum int64

	for _, match := range cpuScalingFiles {
		value, err := readLineAndParseInt(match)
		check(err)
		sum += value
	}

	channel.Put(telem.NewTelemetry("freq", float64(sum)))
}

func (LoadInstrument) MeasureAndReport(channel telem.TelemetryChannel) {
	text, err := readFirstLine("/proc/loadavg")
	check(err)

	parts := strings.Split(text, " ")

	l1 := parts[0]
	l5 := parts[1]
	//l15 := parts[2]

	if val, err := strconv.ParseFloat(l1, 64); err == nil {
		channel.Put(telem.NewTelemetry("load1", val))
	}

	if val, err := strconv.ParseFloat(l5, 64); err == nil {
		channel.Put(telem.NewTelemetry("load5", val))
	}

	//if val, err := strconv.ParseFloat(l15, 64); err == nil {
	//	channel.Put(NewTelemetry("load15", val))
	//}

}

func (instr NetworkDataRateInstrument) MeasureAndReport(channel telem.TelemetryChannel) {
	var wg sync.WaitGroup
	wg.Add(len(instr.Devices))
	defer wg.Wait()

	measureAndReport := func(device string) {
		rxPath := "/sys/class/net/" + device + "/statistics/rx_bytes"
		txPath := "/sys/class/net/" + device + "/statistics/tx_bytes"

		rxThen, err := readLineAndParseInt(rxPath)
		check(err)
		txThen, err := readLineAndParseInt(txPath)
		check(err)

		time.Sleep(1 * time.Second)

		rxNow, err := readLineAndParseInt(rxPath)
		check(err)
		txNow, err := readLineAndParseInt(txPath)
		check(err)

		channel.Put(telem.NewTelemetry("tx"+telem.TopicSeparator+device, float64((txNow-txThen)/1000)))
		channel.Put(telem.NewTelemetry("rx"+telem.TopicSeparator+device, float64((rxNow-rxThen)/1000)))
		wg.Done()
	}

	for _, device := range instr.Devices {
		go measureAndReport(device)
	}
}

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
func readBlockDeviceStats(dev string) []int64 {
	path := "/sys/block/" + dev + "/stat"

	line, err := readFirstLine(path)
	check(err)

	values, err := parseInt64Array(strings.Fields(line))
	check(err)
	return values
}

const sectorSize = 512

func (instr DiskDataRateInstrument) MeasureAndReport(channel telem.TelemetryChannel) {
	var wg sync.WaitGroup
	wg.Add(len(instr.Devices))
	defer wg.Wait()

	measureAndReport := func(device string) {
		defer wg.Done()

		statsThen := readBlockDeviceStats(device)
		time.Sleep(1 * time.Second)
		statsNow := readBlockDeviceStats(device)

		rd := (statsNow[2] - statsThen[2]) * sectorSize
		wr := (statsNow[6] - statsThen[6]) * sectorSize

		channel.Put(telem.NewTelemetry("rd"+telem.TopicSeparator+device, float64(rd)/1000))
		channel.Put(telem.NewTelemetry("wr"+telem.TopicSeparator+device, float64(wr)/1000))
	}

	for _, device := range instr.Devices {
		go measureAndReport(device)
	}
}

type defaultInstrumentFactory struct{}

func (d defaultInstrumentFactory) NewCpuFrequencyInstrument() Instrument {
	return CpuScalingFrequencyInstrument{}
}

func (d defaultInstrumentFactory) NewCpuUtilInstrument() Instrument {
	return CpuUtilInstrument{}
}

func (d defaultInstrumentFactory) NewLoadInstrument() Instrument {
	return LoadInstrument{}
}

func (d defaultInstrumentFactory) NewNetworkDataRateInstrument(devices []string) Instrument {
	return NetworkDataRateInstrument{devices}
}

func (d defaultInstrumentFactory) NewDiskDataRateInstrument(devices []string) Instrument {
	return DiskDataRateInstrument{devices}
}

type armInstrumentFactory struct {
	defaultInstrumentFactory
}

type x86InstrumentFactory struct {
	defaultInstrumentFactory
}

func NewInstrumentFactory(arch string) InstrumentFactory {
	switch arch {
	case "amd64":
		return x86InstrumentFactory{}
	case "arm":
	case "arm64":
		return armInstrumentFactory{}
	default:
		log.Printf("Unknown arch %s, returning default factory", arch)
	}

	return defaultInstrumentFactory{}
}

func parseInt64Array(arr []string) ([]int64, error) {
	ints := make([]int64, len(arr))
	var err error = nil

	for i := 0; i < len(arr); i++ {
		ints[i], err = strconv.ParseInt(arr[i], 10, 64)
		if err != nil {
			return ints, err
		}
	}

	return ints, err
}

// readFirstLine reads and returns the first line from the given file.
// propagates errors from os open and bufio.Scanner.
func readFirstLine(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}

	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)

	scanner.Scan()
	text := scanner.Text()

	return text, scanner.Err()
}

func readLineAndParseInt(path string) (int64, error) {
	line, err := readFirstLine(path)
	if err != nil {
		return -1, err
	}
	return strconv.ParseInt(line, 10, 64)
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
