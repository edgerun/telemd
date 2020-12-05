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
	NewProcsInstrument() Instrument
	NewRamInstrument() Instrument
	NewNetworkDataRateInstrument([]string) Instrument
	NewDiskDataRateInstrument([]string) Instrument
	NewDockerCgroupCpuInstrument() Instrument
	NewKubernetesCgroupCpuInstrument() Instrument
	NewDockerCgroupBlkioInstrument() Instrument
	NewDockerCgroupNetworkInstrument(string) Instrument
	NewDockerCgroupMemoryInstrument() Instrument
	NewKubernetesCgroupBlkioInstrument() Instrument
	NewKubernetesCgroupMemoryInstrument() Instrument
	NewKubernetesCgroupNetInstrument(string) Instrument
	NewPsiCpuInstrument() Instrument
	NewPsiMemoryInstrument() Instrument
	NewPsiIoInstrument() Instrument
	NewWifiTxBitrateInstrument(string) Instrument
	NewWifiRxBitrateInstrument(string) Instrument
	NewWifiSignalInstrument(string) Instrument
	NewGpuFrequencyInstrument(map[int]string) Instrument
	NewGpuUtilInstrument(map[int]string) Instrument
}

type CpuInfoFrequencyInstrument struct{}
type CpuScalingFrequencyInstrument struct{}
type CpuUtilInstrument struct{}
type LoadInstrument struct{}
type ProcsInstrument struct{}
type RamInstrument struct{}
type PsiCpuInstrument struct{}
type PsiMemoryInstrument struct{}
type PsiIoInstrument struct{}
type WifiRxBitrateInstrument struct {
	Device string
}
type WifiTxBitrateInstrument struct {
	Device string
}
type WifiSignalInstrument struct {
	Device string
}
type NetworkDataRateInstrument struct {
	Devices []string
}
type DiskDataRateInstrument struct {
	Devices []string
}
type DockerCgroupv1CpuInstrument struct{}
type DockerCgroupv2CpuInstrument struct{}
type DockerCgroupv1BlkioInstrument struct{}
type DockerCgroupv2BlkioInstrument struct{}
type DockerCgroupv1NetworkInstrument struct {
	pids      map[string]string
	procMount string
}
type DockerCgroupv2NetworkInstrument struct {
	pids      map[string]string
	procMount string
}

type DockerCgroupv1MemoryInstrument struct{}
type DockerCgroupv2MemoryInstrument struct{}

type KubernetesCgroupCpuInstrument struct{}
type KubernetesCgroupBlkioInstrument struct{}
type KuberenetesCgroupMemoryInstrument struct{}
type KubernetesCgroupv1NetworkInstrument struct {
	pids      map[string]string
	procMount string
}

var DisabledInstrument = noopInstrument{}

type noopInstrument struct{}

func (noopInstrument) MeasureAndReport(telem.TelemetryChannel) {
	// noop
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

func (ProcsInstrument) MeasureAndReport(channel telem.TelemetryChannel) {
	text, err := readFirstLine("/proc/loadavg")
	check(err)

	fields := strings.Split(text, " ")

	procs := strings.Split(fields[3], "/")[0]

	if val, err := strconv.ParseFloat(procs, 64); err == nil {
		channel.Put(telem.NewTelemetry("procs", val))
	}
}

func (instr *NetworkDataRateInstrument) MeasureAndReport(channel telem.TelemetryChannel) {
	var wg sync.WaitGroup
	wg.Add(len(instr.Devices))
	defer wg.Wait()

	measureAndReport := func(device string) {
		rxPath := "/sys/class/net/" + device + "/statistics/rx_bytes"
		txPath := "/sys/class/net/" + device + "/statistics/tx_bytes"

		rxThen, err := readLineAndParseInt(rxPath)
		if err != nil {
			log.Println("error while reading path", rxPath, err)
			return
		}
		txThen, err := readLineAndParseInt(txPath)
		if err != nil {
			log.Println("error while reading path", txPath, err)
			return
		}
		time.Sleep(1 * time.Second)

		rxNow, err := readLineAndParseInt(rxPath)
		if err != nil {
			log.Println("error while reading path", rxPath, err)
			return
		}
		txNow, err := readLineAndParseInt(txPath)
		if err != nil {
			log.Println("error while reading path", txPath, err)
			return
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

func (instr *DiskDataRateInstrument) MeasureAndReport(channel telem.TelemetryChannel) {
	var wg sync.WaitGroup
	wg.Add(len(instr.Devices))
	defer wg.Wait()

	measureAndReport := func(device string) {
		defer wg.Done()

		statsThen, err := readBlockDeviceStats(device)
		if err != nil {
			log.Println("error reading block device stats", device, err)
			return
		}

		time.Sleep(1 * time.Second)
		statsNow, err := readBlockDeviceStats(device)
		if err != nil {
			log.Println("error reading block device stats", device, err)
			return
		}

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

func (instr Arm64GpuFrequencyInstrument) MeasureAndReport(channel telem.TelemetryChannel) {
	frequencyInHz, err := readJetsonFrequency()
	if err != nil {
		log.Println("Error reading jetson gpu frequency: ", err)
		return
	}

	frequencyInMHz := frequencyInHz / (1_000_000)
	channel.Put(telem.NewTelemetry("gpu_freq"+telem.TopicSeparator+"0", frequencyInMHz))
}

func (instr X86GpuFrequencyInstrument) MeasureAndReport(channel telem.TelemetryChannel) {
	var wg sync.WaitGroup
	wg.Add(len(instr.Devices))
	defer wg.Wait()

	measureAndReport := func(id int) {
		defer wg.Done()

		// gpu_freq already returns MHz
		frequencies, err := execute("gpu_freq", strconv.Itoa(id))
		if err != nil {
			log.Println("Error reading gpu measurements", err)
		}

		if len(frequencies) != 1 {
			log.Println("Expected 1 cpu freqency measurement but were ", len(frequencies))
			return
		}

		//Format: id-name-measure-value
		values := strings.Split(frequencies[0], "-")
		frequency, err := strconv.ParseFloat(values[3], 64)
		if err != nil {
			log.Println("Expected number from gpu frequency, but got: ", values[3])
			return
		}

		channel.Put(telem.NewTelemetry("gpu_freq"+telem.TopicSeparator+strconv.Itoa(id), frequency))
	}

	for id, _ := range instr.Devices {
		go measureAndReport(id)
	}
}

func (instr DefaultGpuFrequencyInstrument) MeasureAndReport(channel telem.TelemetryChannel) {
	// per default no gpu support
}

func (instr DefaultGpuUtilInstrument) MeasureAndReport(channel telem.TelemetryChannel) {
	// per default no gpu support
}

func (instr Arm64GpuUtilInstrument) MeasureAndReport(channel telem.TelemetryChannel) {
	gpuUtil, err := readJetsonGpuUtilization()
	if err != nil {
		log.Println("Error reading jetson gpu frequency: ", err)
		return
	}

	channel.Put(telem.NewTelemetry("gpu_util"+telem.TopicSeparator+"0", gpuUtil))
}

func (instr X86GpuUtilInstrument) MeasureAndReport(channel telem.TelemetryChannel) {
	var wg sync.WaitGroup
	wg.Add(len(instr.Devices))
	defer wg.Wait()

	measureAndReport := func(id int) {
		defer wg.Done()

		// gpu_util already returns percentage
		frequencies, err := execute("gpu_util", strconv.Itoa(id))
		if err != nil {
			log.Println("Error reading gpu utilization", err)
		}

		if len(frequencies) != 1 {
			log.Println("Expected 1 gpu utilization measurement but were ", len(frequencies))
			return
		}

		//Format: id-name-measure-value
		values := strings.Split(frequencies[0], "-")
		frequency, err := strconv.ParseFloat(values[3], 64)
		if err != nil {
			log.Println("Expected number from gpu_util, but got: ", values[3])
			return
		}

		channel.Put(telem.NewTelemetry("gpu_util"+telem.TopicSeparator+strconv.Itoa(id), frequency))
	}

	for id, _ := range instr.Devices {
		go measureAndReport(id)
	}
}

func (PsiCpuInstrument) MeasureAndReport(channel telem.TelemetryChannel) {
	result, err := readPsiResult("cpu")
	if err == nil {
		channel.Put(telem.NewTelemetry("psi_cpu/some", result.Some.Total))
		if result.Full != nil {
			channel.Put(telem.NewTelemetry("psi_cpu/full", result.Full.Total))
		}
	}
}

func (PsiMemoryInstrument) MeasureAndReport(channel telem.TelemetryChannel) {
	result, err := readPsiResult("memory")
	if err == nil {
		channel.Put(telem.NewTelemetry("psi_memory/some", result.Some.Total))
		if result.Full != nil {
			channel.Put(telem.NewTelemetry("psi_memory/full", result.Full.Total))
		}
	}
}

func (PsiIoInstrument) MeasureAndReport(channel telem.TelemetryChannel) {
	result, err := readPsiResult("io")
	if err == nil {
		channel.Put(telem.NewTelemetry("psi_io/some", result.Some.Total))
		if result.Full != nil {
			channel.Put(telem.NewTelemetry("psi_io/full", result.Full.Total))
		}
	}
}

func (i WifiTxBitrateInstrument) MeasureAndReport(channel telem.TelemetryChannel) {
	bitrate, err := parseIw(i.Device, "tx bitrate", 3)
	if err == nil {
		value, err := strconv.ParseFloat(bitrate, 64)
		if err == nil {
			channel.Put(telem.NewTelemetry("tx_bitrate"+telem.TopicSeparator+i.Device, value))
		}
	}
}

func (i WifiRxBitrateInstrument) MeasureAndReport(channel telem.TelemetryChannel) {
	bitrate, err := parseIw(i.Device, "rx bitrate", 3)
	if err == nil {
		value, err := strconv.ParseFloat(bitrate, 64)
		if err == nil {
			channel.Put(telem.NewTelemetry("rx_bitrate"+telem.TopicSeparator+i.Device, value))
		}
	}
}

func (i WifiSignalInstrument) MeasureAndReport(channel telem.TelemetryChannel) {
	bitrate, err := parseIw(i.Device, "signal", 2)
	if err == nil {
		value, err := strconv.ParseFloat(bitrate, 64)
		if err == nil {
			channel.Put(telem.NewTelemetry("signal"+telem.TopicSeparator+i.Device, value))
		}
	}
}

func (DockerCgroupv1CpuInstrument) MeasureAndReport(channel telem.TelemetryChannel) {
	dirs, err := listFilterDir("/sys/fs/cgroup/cpuacct/docker", func(info os.FileInfo) bool {
		return info.IsDir() && info.Name() != "." && info.Name() != ".."
	})

	if err != nil {
		log.Println("error reading docker cgroup cpu", err)
		return
	}

	for _, containerId := range dirs {
		containerFolder := "/sys/fs/cgroup/cpuacct/docker/" + containerId
		value, err := readCgroupCpu(containerFolder)
		if err == nil {
			channel.Put(telem.NewTelemetry("docker_cgrp_cpu/"+containerId[:12], float64(value)))
		} else {
			log.Println("error reading data file", containerFolder, err)
		}
	}
}

func listCgroupv2Folders(prefix string) (error, []string) {
	dirname := "/sys/fs/cgroup/system.slice"
	dirs, err := listFilterDir(dirname, func(info os.FileInfo) bool {
		return info.IsDir() && info.Name() != "." && info.Name() != ".." && strings.HasPrefix(info.Name(), prefix)
	})

	if err != nil {
		log.Println("error cgroup v2 folders", err)
		return err, make([]string, 0)
	}
	return nil, dirs
}

func (DockerCgroupv2CpuInstrument) MeasureAndReport(channel telem.TelemetryChannel) {
	dirname := "/sys/fs/cgroup/system.slice"
	prefix := "docker-"

	err, dirs := listCgroupv2Folders(prefix)
	if err != nil {
		log.Println(err)
	}
	for _, containerIdFolder := range dirs {
		index := len(prefix)
		containerId := containerIdFolder[index : index+12]
		containerFolder := dirname + "/" + containerIdFolder
		value, err := readCgroupv2Cpu(containerFolder)
		if err == nil {
			channel.Put(telem.NewTelemetry("docker_cgrp_cpu/"+containerId, float64(value)))
		} else {
			log.Println("error reading data file", containerFolder, err)
		}
	}
}
func readCgroupCpu(containerFolder string) (int64, error) {
	dataFile := containerFolder + "/cpuacct.usage"
	value, err := readLineAndParseInt(dataFile)
	if err != nil {
		return -1, err
	}
	return value, nil
}

func readCgroupv2Cpu(containerFolder string) (int64, error) {
	dataFile := containerFolder + "/cpu.stat"
	// usage_usec 293933 (cpu in microseconds)
	value, err := readFirstLine(dataFile)
	if err != nil {
		return -1, err
	}
	unparsed := strings.Split(value, " ")[1]
	parsed, err := strconv.ParseInt(unparsed, 10, 64)
	if err != nil {
		return -1, err
	}
	// convert to nanoseconds - cgroup v1 default
	return parsed * 1000, nil
}

func fetchKubernetesContainerDirs(kubepodDir string) []string {
	if _, err := os.Stat(kubepodDir); os.IsNotExist(err) {
		return make([]string, 0)
	}

	getPods := func(dir string) []string {
		pods, err := listFilterDir(dir, func(info os.FileInfo) bool {
			return info.IsDir() && strings.Contains(info.Name(), "pod")
		})

		if err != nil {
			log.Println("error getting pods", err)
		}

		return pods
	}

	getContainers := func(podDir string) []string {
		containers, err := listFilterDir(podDir, func(info os.FileInfo) bool {
			return info.IsDir() && len(info.Name()) == 64
		})

		if err != nil {
			log.Println("error getting containers", err)
		}

		return containers
	}

	var containerDirs []string
	for _, pod := range getPods(kubepodDir) {
		podDir := kubepodDir + "/" + pod
		for _, containerId := range getContainers(podDir) {
			containerDir := podDir + "/" + containerId
			containerDirs = append(containerDirs, containerDir)
		}
	}
	return containerDirs
}

func (KubernetesCgroupCpuInstrument) MeasureAndReport(channel telem.TelemetryChannel) {
	var kubepodRootDir = "/sys/fs/cgroup/cpuacct/kubepods"
	var bestEffortDir = kubepodRootDir + "/" + "besteffort"
	var burstableDir = kubepodRootDir + "/" + "burstable"
	var guaranteedDir = kubepodRootDir + "/" + "guaranteed"

	for _, kubepodDir := range [3]string{bestEffortDir, burstableDir, guaranteedDir} {
		go func(kubepodDir string) {
			for _, containerDir := range fetchKubernetesContainerDirs(kubepodDir) {
				go func(containerDir string) {
					containerId := filepath.Base(containerDir)
					value, err := readCgroupCpu(containerDir)
					if err == nil {
						log.Println(value)
						channel.Put(telem.NewTelemetry("kubernetes_cgrp_cpu/"+containerId, float64(value)))
					} else {
						log.Println("error reading data file", containerId, err)
					}
				}(containerDir)

			}
		}(kubepodDir)

	}
}

func (c *DockerCgroupv1NetworkInstrument) MeasureAndReport(channel telem.TelemetryChannel) {
	containerIds, err := listFilterDir("/sys/fs/cgroup/cpuacct/docker", func(info os.FileInfo) bool {
		return info.IsDir() && info.Name() != "." && info.Name() != ".."
	})

	if err != nil {
		log.Println("error measuring docker cgroup net", err)
		return
	}
	if len(containerIds) == 0 {
		return
	}

	for _, containerId := range containerIds {
		pid, ok := c.pids[containerId]

		if !ok {
			// refresh pids
			pids, err := containerProcessIds(c.procMount)
			if err != nil {
				log.Println("unable to get container process ids", err)
				continue
			}
			c.pids = pids

			pid, ok = c.pids[containerId]
			if !ok {
				log.Println("could not get pid of container after refresh", containerId)
				continue
			}
		}

		rxValues, txValues, err := readTotalProcessNetworkStats(pid, c.procMount)
		if err != nil {
			if os.IsNotExist(err) {
				delete(c.pids, containerId) // delete now and wait for next iteration to refresh
			} else {
				log.Println("error parsing network stats of pid", pid, err, err)
			}
			continue
		}

		rx := int64(0)
		tx := int64(0)
		for device, irx := range rxValues {
			itx := txValues[device]
			channel.Put(telem.NewTelemetry("docker_cgrp_net/"+containerId+"/"+device, float64(irx+itx)))
			channel.Put(telem.NewTelemetry("docker_cgrp_net/"+containerId+"/"+device+"/rx", float64(irx)))
			channel.Put(telem.NewTelemetry("docker_cgrp_net/"+containerId+"/"+device+"/tx", float64(itx)))
			rx += irx
			tx += itx
		}
		channel.Put(telem.NewTelemetry("docker_cgrp_net/"+containerId, float64(rx+tx)))
	}
}

func (c KubernetesCgroupv1NetworkInstrument) MeasureAndReport(channel telem.TelemetryChannel) {
	var kubepodRootDir = "/sys/fs/cgroup/cpuacct/kubepods"
	var bestEffortDir = kubepodRootDir + "/" + "besteffort"
	var burstableDir = kubepodRootDir + "/" + "burstable"
	var guaranteedDir = kubepodRootDir + "/" + "guaranteed"

	for _, kubepodDir := range [3]string{bestEffortDir, burstableDir, guaranteedDir} {

		go func(kubepodDir string) {
			for _, containerDir := range fetchKubernetesContainerDirs(kubepodDir) {
				containerId := filepath.Base(containerDir)

				pid, ok := c.pids[containerId]

				if !ok {
					// refresh pids
					pids, err := containerProcessIds(c.procMount)
					if err != nil {
						log.Println("unable to get container process ids", err)
						continue
					}
					c.pids = pids

					pid, ok = c.pids[containerId]
					if !ok {
						log.Println("could not get pid of container after refresh", containerId)
						continue
					}
				}

				rxValues, txValues, err := readTotalProcessNetworkStats(pid, c.procMount)
				if err != nil {
					if os.IsNotExist(err) {
						delete(c.pids, containerId) // delete now and wait for next iteration to refresh
					} else {
						log.Println("error parsing network stats of pid", pid, err, err)
					}
					continue
				}

				rx := int64(0)
				tx := int64(0)
				for device, irx := range rxValues {
					itx := txValues[device]
					channel.Put(telem.NewTelemetry("kubernetes_cgrp_net/"+containerId+"/"+device, float64(irx+itx)))
					channel.Put(telem.NewTelemetry("kubernetes_cgrp_net/"+containerId+"/"+device+"/rx", float64(irx)))
					channel.Put(telem.NewTelemetry("kubernetes_cgrp_net/"+containerId+"/"+device+"/tx", float64(itx)))
					rx += irx
					tx += itx
				}
				channel.Put(telem.NewTelemetry("kubernetes_cgrp_net/"+containerId, float64(rx+tx)))
			}
		}(kubepodDir)
	}

}

func (c *DockerCgroupv2NetworkInstrument) MeasureAndReport(channel telem.TelemetryChannel) {
	prefix := "docker-"

	err, dirs := listCgroupv2Folders(prefix)
	if err != nil {
		log.Println("error measuring docker cgroup blkio", err)
		return
	}
	if len(dirs) == 0 {
		return
	}

	for _, containerIdFolder := range dirs {
		index := len(prefix)
		stop := len(containerIdFolder) - len(".scope")
		containerId := containerIdFolder[index:stop]
		pid, ok := c.pids[containerId]

		if !ok {
			// refresh pids
			pids, err := containerProcessIds(c.procMount)
			if err != nil {
				log.Println("unable to get container process ids", err)
				continue
			}
			c.pids = pids

			pid, ok = c.pids[containerId]
			if !ok {
				log.Println("could not get pid of container after refresh", containerId)
				continue
			}
		}

		rxValues, txValues, err := readTotalProcessNetworkStats(pid, c.procMount)
		if err != nil {
			if os.IsNotExist(err) {
				delete(c.pids, containerId) // delete now and wait for next iteration to refresh
			} else {
				log.Println("error parsing network stats of pid", pid, err, err)
			}
			continue
		}

		rx := int64(0)
		tx := int64(0)
		for device, irx := range rxValues {
			itx := txValues[device]
			channel.Put(telem.NewTelemetry("docker_cgrp_net/"+containerId+"/"+device, float64(irx+itx)))
			channel.Put(telem.NewTelemetry("docker_cgrp_net/"+containerId+"/"+device+"/rx", float64(irx)))
			channel.Put(telem.NewTelemetry("docker_cgrp_net/"+containerId+"/"+device+"/tx", float64(itx)))
			rx += irx
			tx += itx
		}
		channel.Put(telem.NewTelemetry("docker_cgrp_net/"+containerId, float64(rx+tx)))
	}
}

func readBlkioTotal(path string) (val int64, err error) {
	visitorErr := visitLines(path, func(line string) bool {
		if strings.HasPrefix(line, "Total") {
			val, err = strconv.ParseInt(strings.Split(line, " ")[1], 10, 64)
			return false
		}
		return true
	})

	if visitorErr != nil {
		return val, visitorErr
	}

	return
}

func (DockerCgroupv1BlkioInstrument) MeasureAndReport(channel telem.TelemetryChannel) {
	dirs, err := listFilterDir("/sys/fs/cgroup/blkio/docker", func(info os.FileInfo) bool {
		return info.IsDir() && info.Name() != "." && info.Name() != ".."
	})

	if err != nil {
		log.Println("error measuring docker cgroup blkio", err)
		return
	}

	for _, containerId := range dirs {
		containerDir := "/sys/fs/cgroup/blkio/docker/" + containerId
		value, err := readCgroupBlkio(containerDir)
		if err != nil {
			log.Println("error reading data file", containerDir, err)
			continue
		}
		channel.Put(telem.NewTelemetry("docker_cgrp_blkio/"+containerId[:12], float64(value)))
	}
}

func (k DockerCgroupv1MemoryInstrument) MeasureAndReport(ch telem.TelemetryChannel) {
	dirs, err := listFilterDir("/sys/fs/cgroup/memory/docker", func(info os.FileInfo) bool {
		return info.IsDir() && info.Name() != "." && info.Name() != ".."
	})

	if err != nil {
		log.Println("error measuring docker cgroup memory", err)
		return
	}

	for _, containerId := range dirs {
		containerDir := "/sys/fs/cgroup/memory/docker/" + containerId
		value, err := readCgroupMemory(containerDir)
		if err == nil {
			ch.Put(telem.NewTelemetry("docker_cgrp_memory/"+containerId, float64(value)))
		} else {
			log.Println("error reading data file", containerId, err)
		}
	}
}

func readCgroupBlkio(containerDir string) (int64, error) {
	dataFile := containerDir + "/blkio.throttle.io_service_bytes"
	value, err := readBlkioTotal(dataFile)
	if err != nil {
		return -1, err
	}
	return value, nil
}

func readIoStat(path string) (int64, int64, error) {
	// contains multiple lines
	// each line looks similar to:
	// 259:0 rbytes=10113024 wbytes=0 rios=149 wios=0 dbytes=0 dios=0
	// each rbytes and wbytes must be summed up

	file, err := os.Open(path)
	if err != nil {
		return -1, -1, err
	}
	defer file.Close()

	readBytes := int64(0)
	writeBytes := int64(0)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		rawReadBytes := strings.Split(line, " ")[1]
		rawReadBytes = strings.Split(rawReadBytes, "=")[1]

		rawWriteBytes := strings.Split(line, " ")[2]
		rawWriteBytes = strings.Split(rawWriteBytes, "=")[1]

		parsedReadBytes, err := parseInt64(rawReadBytes)
		if err != nil {
			return -1, -1, err
		}

		parsedWriteBytes, err := parseInt64(rawWriteBytes)
		if err != nil {
			return -1, -1, err
		}

		readBytes += parsedReadBytes
		writeBytes += parsedWriteBytes
	}

	return readBytes, writeBytes, nil
}

func readCgroupv2Blkio(containerDir string) (int64, error) {
	dataFile := containerDir + "/io.stat"
	readBytes, writeBytes, err := readIoStat(dataFile)
	if err != nil {
		return -1, err
	}
	return readBytes + writeBytes, nil
}

func (DockerCgroupv2BlkioInstrument) MeasureAndReport(channel telem.TelemetryChannel) {
	dirname := "/sys/fs/cgroup/system.slice"
	prefix := "docker-"

	err, dirs := listCgroupv2Folders(prefix)
	if err != nil {
		log.Println("error measuring docker cgroup blkio", err)
		return
	}

	for _, containerIdFolder := range dirs {
		index := len(prefix)
		// 12 is the length of the short-form for container IDs
		containerId := containerIdFolder[index : index+12]
		containerFolder := dirname + "/" + containerIdFolder
		value, err := readCgroupv2Blkio(containerFolder)
		if err != nil {
			log.Println("error reading data file", containerFolder, err)
			continue
		}
		channel.Put(telem.NewTelemetry("docker_cgrp_blkio/"+containerId, float64(value)))
	}
}

func (k DockerCgroupv2MemoryInstrument) MeasureAndReport(ch telem.TelemetryChannel) {
	dirname := "/sys/fs/cgroup/system.slice"
	prefix := "docker-"

	err, dirs := listCgroupv2Folders(prefix)
	if err != nil {
		log.Println("error measuring docker cgroup blkio", err)
		return
	}

	for _, containerIdFolder := range dirs {
		index := len(prefix)
		// 12 is the length of the short-form for container IDs
		containerId := containerIdFolder[index : index+12]
		containerFolder := dirname + "/" + containerIdFolder
		value, err := readCgroupv2Memory(containerFolder)
		if err == nil {
			ch.Put(telem.NewTelemetry("docker_cgrp_memory/"+containerId, float64(value)))
		} else {
			log.Println("error reading data file", containerId, err)
		}
	}
}

func (KubernetesCgroupBlkioInstrument) MeasureAndReport(channel telem.TelemetryChannel) {
	var kubepodRootDir = "/sys/fs/cgroup/blkio/kubepods"
	var bestEffortDir = kubepodRootDir + "/" + "besteffort"
	var burstableDir = kubepodRootDir + "/" + "burstable"
	var guaranteedDir = kubepodRootDir + "/" + "guaranteed"

	for _, kubepodDir := range [3]string{bestEffortDir, burstableDir, guaranteedDir} {
		for _, containerDir := range fetchKubernetesContainerDirs(kubepodDir) {
			containerId := filepath.Base(containerDir)
			value, err := readCgroupBlkio(containerDir)
			if err == nil {
				channel.Put(telem.NewTelemetry("kubernetes_cgrp_blkio/"+containerId, float64(value)))
			} else {
				log.Println("error reading data file", containerId, err)
			}
		}
	}
}

func readMemory(path string) (val int64, err error) {
	visitorErr := visitLines(path, func(line string) bool {
		val, err = strconv.ParseInt(line, 10, 64)
		return true
	})
	if visitorErr != nil {
		return val, visitorErr
	}
	return
}

func readCgroupMemory(containerDir string) (int64, error) {
	dataFile := containerDir + "/memory.usage_in_bytes"
	value, err := readMemory(dataFile)
	if err != nil {
		return -1, err
	}
	return value, nil
}

func readCgroupv2Memory(containerDir string) (int64, error) {
	dataFile := containerDir + "/memory.current"
	value, err := readLineAndParseInt(dataFile)
	if err != nil {
		return -1, err
	}
	return value, nil
}

func (k KuberenetesCgroupMemoryInstrument) MeasureAndReport(ch telem.TelemetryChannel) {
	var kubepodRootDir = "/sys/fs/cgroup/memory/kubepods"
	var bestEffortDir = kubepodRootDir + "/" + "besteffort"
	var burstableDir = kubepodRootDir + "/" + "burstable"
	var guaranteedDir = kubepodRootDir + "/" + "guaranteed"

	for _, kubePodDir := range [3]string{bestEffortDir, burstableDir, guaranteedDir} {
		for _, containerDir := range fetchKubernetesContainerDirs(kubePodDir) {
			containerId := filepath.Base(containerDir)
			value, err := readCgroupMemory(containerDir)
			if err == nil {
				ch.Put(telem.NewTelemetry("kubernetes_cgrp_memory/"+containerId, float64(value)))
			} else {
				log.Println("error reading data file", containerId, err)
			}
		}
	}
}

type defaultInstrumentFactory struct{}

type arm32InstrumentFactory struct {
	defaultInstrumentFactory
}

type arm64InstrumentFactory struct {
	defaultInstrumentFactory
}

type x86InstrumentFactory struct {
	defaultInstrumentFactory
}

func (d defaultInstrumentFactory) NewCpuFrequencyInstrument() Instrument {
	return CpuScalingFrequencyInstrument{}
}

func (d defaultInstrumentFactory) NewCpuUtilInstrument() Instrument {
	return CpuUtilInstrument{}
}

func (d defaultInstrumentFactory) NewLoadInstrument() Instrument {
	return LoadInstrument{}
}

func (d defaultInstrumentFactory) NewProcsInstrument() Instrument {
	return ProcsInstrument{}
}

func (d defaultInstrumentFactory) NewRamInstrument() Instrument {
	return RamInstrument{}
}

func (d defaultInstrumentFactory) NewNetworkDataRateInstrument(devices []string) Instrument {
	return &NetworkDataRateInstrument{devices}
}

func (d defaultInstrumentFactory) NewDiskDataRateInstrument(devices []string) Instrument {
	return &DiskDataRateInstrument{devices}
}

func (d defaultInstrumentFactory) NewWifiTxBitrateInstrument(device string) Instrument {
	return &WifiTxBitrateInstrument{device}
}

func (d defaultInstrumentFactory) NewWifiRxBitrateInstrument(device string) Instrument {
	return &WifiRxBitrateInstrument{device}
}

func (d defaultInstrumentFactory) NewWifiSignalInstrument(device string) Instrument {
	return &WifiSignalInstrument{device}
}

func (d defaultInstrumentFactory) NewPsiCpuInstrument() Instrument {
	return PsiCpuInstrument{}
}

func (d defaultInstrumentFactory) NewPsiMemoryInstrument() Instrument {
	return PsiMemoryInstrument{}
}

func (d defaultInstrumentFactory) NewPsiIoInstrument() Instrument {
	return PsiIoInstrument{}
}

func checkCgroup() string {
	if _, err := os.Stat("/sys/fs/cgroup/cgroup.controllers"); os.IsNotExist(err) {
		return "v1"
	} else {
		return "v2"
	}
}

func (d defaultInstrumentFactory) NewDockerCgroupCpuInstrument() Instrument {
	cgroup := checkCgroup()
	if cgroup == "v1" {
		return DockerCgroupv1CpuInstrument{}
	} else {
		return DockerCgroupv2CpuInstrument{}
	}

}

func (d defaultInstrumentFactory) NewKubernetesCgroupCpuInstrument() Instrument {
	return KubernetesCgroupCpuInstrument{}
}

func (d defaultInstrumentFactory) NewDockerCgroupBlkioInstrument() Instrument {
	cgroup := checkCgroup()
	if cgroup == "v1" {
		return DockerCgroupv1BlkioInstrument{}
	} else {
		return DockerCgroupv2BlkioInstrument{}
	}
}

func (d defaultInstrumentFactory) NewDockerCgroupNetworkInstrument(procMount string) Instrument {
	pidMap, err := containerProcessIds(procMount)

	if err != nil {
		log.Println("unable to get process ids of containers", err)
	}
	cgroup := checkCgroup()

	if cgroup == "v1" {
		return &DockerCgroupv1NetworkInstrument{
			pids:      pidMap,
			procMount: procMount,
		}
	} else {
		return &DockerCgroupv2NetworkInstrument{
			pids:      pidMap,
			procMount: procMount,
		}
	}

}

func (d defaultInstrumentFactory) NewKubernetesCgroupBlkioInstrument() Instrument {
	return KubernetesCgroupBlkioInstrument{}
}

func (d defaultInstrumentFactory) NewKubernetesCgroupMemoryInstrument() Instrument {
	return KuberenetesCgroupMemoryInstrument{}
}
func (d defaultInstrumentFactory) NewKubernetesCgroupNetInstrument(procMount string) Instrument {
	pidMap, err := containerProcessIds(procMount)

	if err != nil {
		log.Println("unable to get process ids of containers", err)
	}

	return KubernetesCgroupv1NetworkInstrument{
		pids:      pidMap,
		procMount: procMount,
	}
}

func (d defaultInstrumentFactory) NewDockerCgroupMemoryInstrument() Instrument {
	cgroup := checkCgroup()
	if cgroup == "v1" {
		return DockerCgroupv1MemoryInstrument{}
	} else {
		return DockerCgroupv2MemoryInstrument{}
	}
}

func NewInstrumentFactory(arch string) InstrumentFactory {
	switch arch {
	case "amd64":
		return x86InstrumentFactory{}
	case "arm":
		return arm32InstrumentFactory{}
	case "arm64":
		return arm64InstrumentFactory{}
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
