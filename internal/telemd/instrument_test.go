package telemd

import (
	"github.com/edgerun/go-telemd/internal/telem"
	"testing"
)

func TestReadBlockDeviceStats(t *testing.T) {
	stats := readBlockDeviceStats("loop0")

	if len(stats) < 15 {
		t.Error("Expected at least 15 stats values")
	}

	for i := range stats {
		if stats[i] != 0 {
			t.Error("Expected 0 on loop0")
		}
	}
}

func TestDiskDataRateInstrument_MeasureAndReport(t *testing.T) {
	tc := telem.NewTelemetryChannel()
	instrument := DiskDataRateInstrument{[]string{"loop0"}}

	go instrument.MeasureAndReport(tc)
	ch := tc.Channel()

	t1 := <-ch
	if t1.Value != 0 {
		t.Error("Expected 0 reads on loop0")
	}

	t2 := <-ch
	if t2.Value != 0 {
		t.Error("Expected 0 writes on loop0")
	}

	tc.Close()
}
