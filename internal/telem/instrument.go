package telem

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type CpuFrequencyInstrument struct{}
type CpuUtilInstrument struct{}
type LoadInstrument struct{}
type RamInstrument struct{}
type NetworkDataRateInstrument struct {
	Device string
}
type DiskDataRateInstrument struct {
	Device string
}

// readCpuUtil returns an array of the following values from /proc/stat
// user, nice, system, idle, iowait, irq, softirq
func readCpuUtil() []float64 {
	line, err := ReadFirstLine("/proc/stat")
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

func (CpuUtilInstrument) MeasureAndReport(channel TelemetryChannel) {
	then := readCpuUtil()
	time.Sleep(500 * time.Millisecond)
	now := readCpuUtil()

	val := (now[0] - then[0] + now[2] - then[2]) * 100. / (now[0] - then[0] + now[2] - then[2] + now[3] - then[3])
	channel.Put(NewTelemetry("cpu", val))
}

func (CpuFrequencyInstrument) MeasureAndReport(channel TelemetryChannel) {
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

	channel.Put(NewTelemetry("freq", sum))
}

func (LoadInstrument) MeasureAndReport(channel TelemetryChannel) {
	text, err := ReadFirstLine("/proc/loadavg")
	check(err)

	parts := strings.Split(text, " ")

	l1 := parts[0]
	l5 := parts[1]
	//l15 := parts[2]

	if val, err := strconv.ParseFloat(l1, 64); err == nil {
		channel.Put(NewTelemetry("load1", val))
	}

	if val, err := strconv.ParseFloat(l5, 64); err == nil {
		channel.Put(NewTelemetry("load5", val))
	}

	//if val, err := strconv.ParseFloat(l15, 64); err == nil {
	//	channel.Put(NewTelemetry("load15", val))
	//}

}

func (instr NetworkDataRateInstrument) MeasureAndReport(channel TelemetryChannel) {
	device := instr.Device

	rxPath := "/sys/class/net/" + device + "/statistics/rx_bytes"
	txPath := "/sys/class/net/" + device + "/statistics/tx_bytes"

	rxThen, err := ReadLineAndParseInt(rxPath)
	check(err)
	txThen, err := ReadLineAndParseInt(txPath)
	check(err)

	time.Sleep(1 * time.Second)

	rxNow, err := ReadLineAndParseInt(rxPath)
	check(err)
	txNow, err := ReadLineAndParseInt(txPath)
	check(err)

	channel.Put(NewTelemetry("tx"+TopicSeparator+device, float64((txNow-txThen)/1000)))
	channel.Put(NewTelemetry("rx"+TopicSeparator+device, float64((rxNow-rxThen)/1000)))
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

	line, err := ReadFirstLine(path)
	check(err)

	values, err := ParseInt64Array(strings.Fields(line))
	check(err)
	return values
}

const sectorSize = 512

func (instr DiskDataRateInstrument) MeasureAndReport(channel TelemetryChannel) {
	device := instr.Device

	statsThen := readBlockDeviceStats(device)
	time.Sleep(1 * time.Second)
	statsNow := readBlockDeviceStats(device)

	rd := (statsNow[2] - statsThen[2]) * sectorSize
	wr := (statsNow[6] - statsThen[6]) * sectorSize

	channel.Put(NewTelemetry("rd"+TopicSeparator+device, float64(rd)/1000))
	channel.Put(NewTelemetry("wr"+TopicSeparator+device, float64(wr)/1000))
}

type defaultInstrumentFactory struct{}

func (d defaultInstrumentFactory) NewCpuFrequencyInstrument() Instrument {
	return CpuFrequencyInstrument{}
}

func (d defaultInstrumentFactory) NewCpuUtilInstrument() Instrument {
	return CpuUtilInstrument{}
}

func (d defaultInstrumentFactory) NewLoadInstrument() Instrument {
	return LoadInstrument{}
}

func (d defaultInstrumentFactory) NewNetworkDataRateInstrument(device string) Instrument {
	return NetworkDataRateInstrument{device}
}

func (d defaultInstrumentFactory) NewDiskDataRateInstrument(device string) Instrument {
	return DiskDataRateInstrument{device}
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
