package env

import (
	"os"
	"time"
)

var OsEnv = &osEnvironment{}

type osEnvironment struct{}

func (*osEnvironment) Set(key string, value string) {
	err := os.Setenv(key, value)
	if err != nil {
		panic(err)
	}
}

func (*osEnvironment) Lookup(key string) (string, bool) {
	return os.LookupEnv(key)
}

func (*osEnvironment) Get(key string) string {
	return os.Getenv(key)
}

func (env *osEnvironment) LookupInt(key string) (int64, bool, error) {
	return LookupInt(env, key)
}

func (env *osEnvironment) LookupFloat(key string) (float64, bool, error) {
	return LookupFloat(env, key)
}

func (env *osEnvironment) LookupFields(key string) ([]string, bool, error) {
	return LookupFields(env, key)
}

func (env *osEnvironment) LookupBool(key string) (bool, bool, error) {
	return LookupBool(env, key)
}

func (env *osEnvironment) LookupDuration(key string) (time.Duration, bool, error) {
	return LookupDuration(env, key)
}
