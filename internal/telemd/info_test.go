package telemd

import "testing"

func TestReadSysInfo(t *testing.T) {
	var info NodeInfo
	err := ReadSysInfo(&info)

	if err != nil {
		t.Error("Unexpected error while reading sys info", err)
	}

	info.Print()
}
