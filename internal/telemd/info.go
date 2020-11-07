package telemd

import (
	"fmt"
	"os"
	"runtime"
)

type NodeInfo struct {
	Arch     string
	Cpus     int
	Ram      int64
	Boot     int64
	Disk     []string
	Net      []string
	Hostname string
	EthernetSpeed string
}

func (info NodeInfo) Print() {
	fmt.Println("Arch:     ", info.Arch)
	fmt.Println("Cpus:     ", info.Cpus)
	fmt.Println("Ram:      ", info.Ram)
	fmt.Println("Boot:     ", info.Boot)
	fmt.Println("Disk:     ", info.Disk)
	fmt.Println("Net:      ", info.Net)
	fmt.Println("Hostname: ", info.Hostname)
	fmt.Println("EthernetSpeed: ", info.EthernetSpeed)
}

func SysInfo() NodeInfo {
	var info NodeInfo

	err := ReadSysInfo(&info)
	if err != nil {
		panic(err)
	}

	return info
}

func ReadSysInfo(info *NodeInfo) error {
	info.Arch = runtime.GOARCH
	info.Cpus = runtime.NumCPU()

	ram, err := readMemTotal()
	if err != nil {
		return err
	}
	info.Ram = ram

	boot, err := bootTime()
	if err != nil {
		return err
	}
	info.Boot = boot

	info.Disk = blockDevices()
	info.Net = networkDevices()

	hostname, err := os.Hostname()
	if err != nil {
		return err
	}
	info.Hostname = hostname
	info.EthernetSpeed = ethernetSpeed()

	return nil
}
