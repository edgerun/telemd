package env

import (
	"testing"
	"time"
)

func TestIniEnvironment_Lookup(t *testing.T) {
	env, err := NewIniEnvironment("../../testfiles/test_ini.ini")

	if err != nil {
		t.Error("Error parsing ini", err)
		t.FailNow()
	}

	lookup, ok := env.Lookup("app_mode")

	if !ok {
		t.Error("Expected environment to have 'app_mode'")
		t.FailNow()
	}

	if lookup != "development" {
		t.Error("Expected value: development, actual: ", lookup)
	}
}

func TestIniSectionEnvironment_Lookup(t *testing.T) {
	env, err := NewIniSectionEnvironment("../../testfiles/test_ini.ini", "server")

	if err != nil {
		t.Error("Error parsing ini", err)
		t.FailNow()
	}

	lookup, ok := env.Lookup("protocol")
	if !ok {
		t.Error("Expected environment to have 'app_mode'")
		t.FailNow()
	}
	if lookup != "http" {
		t.Error("Expected value: http, actual: ", lookup)
		t.FailNow()
	}

	intLookup, ok, err := env.LookupInt("http_port")
	if !ok {
		t.Error("Expected environment to have 'app_mode'")
		t.FailNow()
	}
	if intLookup != 9999 {
		t.Error("Expected value: 9999, actual: ", intLookup)
		t.FailNow()
	}

	duration1Lookup, ok, err := env.LookupDuration("duration1")
	if !ok {
		t.Error("Expected environment to have 'duration1'")
		t.FailNow()
	}
	if duration1Lookup != 100*time.Millisecond {
		t.Error("Expected value: 100ms, actual: ", duration1Lookup)
		t.FailNow()
	}

	duration2Lookup, ok, err := env.LookupDuration("duration2")
	if !ok {
		t.Error("Expected environment to have 'duration2'")
		t.FailNow()
	}
	if duration2Lookup != 1*time.Second {
		t.Error("Expected value: 1s, actual: ", duration2Lookup)
		t.FailNow()
	}
}
