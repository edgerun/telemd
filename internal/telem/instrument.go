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
	Iface string
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
	text, err := readFirstLine("/proc/loadavg")
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

func readLineAndParseInt(path string) (int64, error) {
	line, err := readFirstLine(path)
	if err != nil {
		return -1, err
	}
	return strconv.ParseInt(line, 10, 64)
}

func (instr NetworkDataRateInstrument) MeasureAndReport(channel TelemetryChannel) {
	iface := instr.Iface

	rxPath := "/sys/class/net/" + iface + "/statistics/rx_bytes"
	txPath := "/sys/class/net/" + iface + "/statistics/tx_bytes"

	rxThen, err := readLineAndParseInt(rxPath)
	check(err)
	txThen, err := readLineAndParseInt(txPath)
	check(err)

	time.Sleep(1 * time.Second)

	rxNow, err := readLineAndParseInt(rxPath)
	check(err)
	txNow, err := readLineAndParseInt(txPath)
	check(err)

	channel.Put(NewTelemetry("tx", float64((txNow-txThen)/1000)))
	channel.Put(NewTelemetry("rx", float64((rxNow-rxThen)/1000)))
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

func (d defaultInstrumentFactory) NewNetworkDataRateInstrument(iface string) Instrument {
	return NetworkDataRateInstrument{iface}
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
