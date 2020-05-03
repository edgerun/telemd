package telem

import "testing"

func TestParseInt64Array(t *testing.T) {
	arr := []string{"4", "2", "0"}

	ints, err := ParseInt64Array(arr)
	if err != nil {
		t.Error("Unexpected error")
	}

	if len(ints) != 3 {
		t.Error("Unexpected array length (expected 3)", len(ints))
	}

	if ints[0] != 4 {
		t.Error("Unexpected value at 0:", ints[0])
	}
	if ints[1] != 2 {
		t.Error("Unexpected value at 1:", ints[1])
	}
	if ints[2] != 0 {
		t.Error("Unexpected value at 2:", ints[2])
	}
}
