package env

import (
	"strconv"
	"strings"
	"time"
)

type Environment interface {
	// Lookup retrieves the value of the environment variable named by the key. If the variable is present in the
	// environment the value (which may be empty) is returned and the boolean is true. Otherwise the returned value will
	// be empty and the boolean will be false
	Lookup(key string) (string, bool)

	// Get retrieves the value of the environment variable named by the key. It returns the value, which will be empty
	// if the variable is not present. To distinguish between an empty value and an unset value, use Lookup.
	Get(key string) string

	Set(key string, value string)

	LookupInt(key string) (int64, bool, error)
	LookupFloat(key string) (float64, bool, error)
	LookupBool(key string) (bool, bool, error)
	LookupFields(key string) ([]string, bool, error)
	LookupDuration(key string) (time.Duration, bool, error)
}

func LookupInt(env Environment, key string) (int64, bool, error) {
	if value, ok := env.Lookup(key); ok {
		v, err := strconv.ParseInt(value, 10, 64)
		return v, true, err
	} else {
		return 0, false, nil
	}
}

func LookupFloat(env Environment, key string) (float64, bool, error) {
	if value, ok := env.Lookup(key); ok {
		v, err := strconv.ParseFloat(value, 64)
		return v, true, err
	} else {
		return 0, false, nil
	}
}

func LookupFields(env Environment, key string) ([]string, bool, error) {
	if value, ok := env.Lookup(key); ok {
		return strings.Fields(value), true, nil
	} else {
		return nil, false, nil
	}
}

func LookupBool(env Environment, key string) (bool, bool, error) {
	if value, ok := env.Lookup(key); ok {
		v, err := strconv.ParseBool(value)
		return v, true, err
	} else {
		return false, false, nil
	}
}

func LookupDuration(env Environment, key string) (time.Duration, bool, error) {
	if value, ok := env.Lookup(key); ok {
		v, err := time.ParseDuration(value)
		return v, true, err
	} else {
		return 0, false, nil
	}
}
