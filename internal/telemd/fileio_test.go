package telemd

import (
	"strings"
	"testing"
)

func TestVisitLines(t *testing.T) {
	vst := func(line string) bool {
		println(line)

		if strings.HasPrefix(line, "Active:") {
			return false
		}

		return true
	}

	err := visitLines("/proc/meminfo", vst)

	if err != nil {
		t.Error("Unexpected error", err)
	}
}
