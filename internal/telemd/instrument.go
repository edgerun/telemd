package telemd

import (
	"bufio"
	"github.com/edgerun/telemd/internal/telem"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"os/exec"
	"fmt"
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
	NewRamInstrument() Instrument
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

		if strings.HasPrefix(device, "e"){
                        speedPath := "/sys/class/net/" + device + "/speed"
                        speed, err := readLineAndParseInt(speedPath)
                        check(err)
                        channel.Put(telem.NewTelemetry("link"+telem.TopicSeparator+device, float64(speed)))
                } else if strings.HasPrefix(device, "w") {
                        speedstr := exec_command(device)
                        speed, err:= strconv.ParseInt(speedstr, 10, 64)
                        check(err)
                        channel.Put(telem.NewTelemetry("link"+telem.TopicSeparator+device, float64(speed)))
                }

		channel.Put(telem.NewTelemetry("tx"+telem.TopicSeparator+device, float64((txNow-txThen)/1000)))
		channel.Put(telem.NewTelemetry("rx"+telem.TopicSeparator+device, float64((rxNow-rxThen)/1000)))
		wg.Done()
	}

	for _, device := range instr.Devices {
		go measureAndReport(device)
	}
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

func (instr RamInstrument) MeasureAndReport(channel telem.TelemetryChannel) {
	meminfo := readMeminfo()

	totalString, ok := meminfo["MemTotal"]
	if !ok {
		return
	}
	total, err := parseMeminfoString(totalString)
	if err != nil {
		log.Println("Error parsing MemTotal string", totalString, err)
	}

	freeString, ok := meminfo["MemAvailable"]
	if !ok {
		return
	}
	free, err := parseMeminfoString(freeString)
	if err != nil {
		log.Println("Error parsing MemFree string", freeString, err)
	}

	channel.Put(telem.NewTelemetry("ram", float64(total-free)))
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

func (d defaultInstrumentFactory) NewRamInstrument() Instrument {
	return RamInstrument{}
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

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func exec_command(device string) string {
        args := "iw dev "+device+" link | awk -F '[ ]' '/tx bitrate:/{print $3}'"
        cmd := exec.Command("sh","-c", args)
        if output,err := cmd.Output(); err!= nil {
                log.Printf( "Error fetching wifi bitrate: %s",err)
        }else{
                log.Printf( "wifi bitrate: %s",output)
                str_output := strings.TrimSpace(string(output))
                value, _ := strconv.ParseFloat(str_output,32)
                return fmt.Sprint(int(value))
        }
        return ""
}

