package telemd

import (
	"fmt"
	"log"
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
	NetSpeed string
}

func (info NodeInfo) Print() {
	fmt.Println("Arch:     ", info.Arch)
	fmt.Println("Cpus:     ", info.Cpus)
	fmt.Println("Ram:      ", info.Ram)
	fmt.Println("Boot:     ", info.Boot)
	fmt.Println("Disk:     ", info.Disk)
	fmt.Println("Net:      ", info.Net)
	fmt.Println("Hostname: ", info.Hostname)
	fmt.Println("netSpeed: ", info.NetSpeed)
}

func SysInfo() NodeInfo {
	var info NodeInfo

	ReadSysInfo(&info)

	return info
}

func ReadSysInfo(info *NodeInfo) {
	info.Arch = runtime.GOARCH
	info.Cpus = runtime.NumCPU()

	if ram, err := readMemTotal(); err == nil {
		info.Ram = ram
	} else {
		log.Println("error reading ram info", err)
	}

	if boot, err := bootTime(); err == nil {
		info.Boot = boot
	} else {
		log.Println("error reading boot time info", err)
	}

	info.Disk = blockDevices()
	info.Net = networkDevices()

	if hostname, err := os.Hostname(); err == nil {
		info.Hostname = hostname
	} else {
		log.Println("error reading hostname info", err)
	}

	if netSpeed, err := netSpeed(); err == nil {
		info.NetSpeed = netSpeed
	} else {
		log.Println("error reading network speed info", err)
	}
}
