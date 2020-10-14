package env

import (
	"gopkg.in/ini.v1"
	"time"
)

type iniEnvironment struct {
	cfg     *ini.File
	section string
}

func NewIniEnvironment(path string) (Environment, error) {
	return NewIniSectionEnvironment(path, "")
}

func NewIniSectionEnvironment(path string, section string) (Environment, error) {
	cfg, err := ini.Load(path)
	if err != nil {
		return nil, err
	}

	env := &iniEnvironment{
		cfg:     cfg,
		section: section,
	}

	return env, nil
}

func (env *iniEnvironment) Set(key string, value string) {
	env.cfg.Section(env.section).Key(key).SetValue(value)
}

func (env *iniEnvironment) Lookup(key string) (value string, ok bool) {
	section := env.cfg.Section(env.section)
	if !section.HasKey(key) {
		return "", false
	}
	return section.Key(key).Value(), true
}

func (env *iniEnvironment) Get(key string) string {
	data, _ := env.Lookup(key)
	return data
}

func (env *iniEnvironment) LookupInt(key string) (int64, bool, error) {
	return LookupInt(env, key)
}

func (env *iniEnvironment) LookupFloat(key string) (float64, bool, error) {
	return LookupFloat(env, key)
}

func (env *iniEnvironment) LookupFields(key string) ([]string, bool, error) {
	return LookupFields(env, key)
}

func (env *iniEnvironment) LookupBool(key string) (bool, bool, error) {
	return LookupBool(env, key)
}

func (env *iniEnvironment) LookupDuration(key string) (time.Duration, bool, error) {
	return LookupDuration(env, key)
}
