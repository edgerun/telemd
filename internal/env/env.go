package env

import (
	"strconv"
	"strings"
)

type Environment interface {
	Lookup(key string) (string, bool)
	Get(key string) string
	Set(key string, value string)

	LookupInt(key string) (int64, bool, error)
	LookupFloat(key string) (float64, bool, error)
	LookupBool(key string) (bool, bool, error)
	LookupFields(key string) ([]string, bool, error)
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
