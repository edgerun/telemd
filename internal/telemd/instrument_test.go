package telemd

import (
	"github.com/edgerun/telemd/internal/telem"
	"log"
	"testing"
	"time"
)

// TODO: proper tests and use timeouts for channel reads

func TestReadBlockDeviceStats(t *testing.T) {
	stats, _ := readBlockDeviceStats("loop0")

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

func TestRamInstrument_MeasureAndReport(t *testing.T) {
	var instrument RamInstrument
	tc := telem.NewTelemetryChannel()

	go instrument.MeasureAndReport(tc)

	t1 := <-tc.Channel()
	if t1.Value <= 0 {
		t.Error("Expected some RAM to be used")
	}
	log.Printf("%.4f\n", t1.Value)
}

func TestX86GpuFrequencyInstrument_MeasureAndReport(t *testing.T) {
	instrument := X86GpuFrequencyInstrument{map[int]string{0: "dummy_gpu"}}
	tc := telem.NewTelemetryChannel()

	go instrument.MeasureAndReport(tc)

	select {
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for GPU frequency")
	case t1 := <-tc.Channel():
		if t1.Value < 0 {
			t.Error("Unexpected negative GPU frequency")
		}
		log.Printf("%.4f\n", t1.Value)
	}

}

func TestLoadInstrument_MeasureAndReport(t *testing.T) {
	var instrument LoadInstrument
	tc := telem.NewTelemetryChannel()

	go instrument.MeasureAndReport(tc)

	t0 := <-tc.Channel()
	if t0.Topic != "load1" {
		t.Error("Expected first value to be from load1")
	}
	log.Printf("%s: %.4f\n", t0.Topic, t0.Value)

	t1 := <-tc.Channel()
	if t1.Topic != "load5" {
		t.Error("Expected first value to be from load1")
	}
	log.Printf("%s: %.4f\n", t1.Topic, t1.Value)

}

func TestProcsInstrument_MeasureAndReport(t *testing.T) {
	var instrument ProcsInstrument
	tc := telem.NewTelemetryChannel()

	go instrument.MeasureAndReport(tc)

	t0 := <-tc.Channel()

	if t0.Topic != "procs" {
		t.Error("Expected value to be from procs")
	}
	if t0.Value <= 0 {
		t.Error("Expected some processes to run")
	}
	log.Printf("%s: %.4f\n", t0.Topic, t0.Value)
}
