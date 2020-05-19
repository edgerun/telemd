package telemd

import "testing"

func TestReadMeminfo(t *testing.T) {
	meminfo := readMeminfo()

	if _, ok := meminfo["MemTotal"]; !ok {
		t.Error("Expected meminfo to have a key 'MemTotal'")
	}

	for k, v := range meminfo {
		println(k, v)
	}
}
