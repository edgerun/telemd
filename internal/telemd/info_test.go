package telemd

import "testing"

func TestReadSysInfo(t *testing.T) {
	var info NodeInfo
	ReadSysInfo(&info)
	info.Print()
}
