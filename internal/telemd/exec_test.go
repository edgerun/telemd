package telemd

import "testing"

func TestExecute(t *testing.T) {
	text := "hello, world"
	response, err := execute("echo", text)

	if err != nil {
		t.Error("Unexpected error", err)
	}

	if len(response) != 1 {
		t.Error("Expected response to only contain a single line but was: ", len(response))
	}

	if text != response[0] {
		t.Error("Response should be '", text, "' but was: ", response)
	}
}
