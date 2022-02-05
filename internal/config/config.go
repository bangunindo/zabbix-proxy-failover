package config

import (
	"io/ioutil"
	"time"

	"github.com/caarlos0/env/v6"
	"github.com/goccy/go-yaml"
)

type Tag struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

type Proxy struct {
	ProxyID   int   `yaml:"proxy_id"`
	Tag       []Tag `yaml:"tag"`
	Weight    int   `yaml:"weight"`
	TriggerID *int  `yaml:"trigger_id"`
}

type Check struct {
	Interval          time.Duration `yaml:"interval"`
	Timeout           time.Duration `yaml:"timeout"`
	ExecutionDuration time.Duration `yaml:"execution_duration"`
}

type Zabbix struct {
	URL      string `yaml:"url"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

type FailoverMethod int

const (
	LASTSEEN FailoverMethod = iota
	TRIGGER
)

var failoverToString = map[FailoverMethod]string{
	LASTSEEN: "last_seen",
	TRIGGER:  "trigger",
}

var failoverToID = map[string]FailoverMethod{
	"last_seen": LASTSEEN,
	"trigger":   TRIGGER,
}

func (f FailoverMethod) MarshalYAML() (interface{}, error) {
	return f.String(), nil
}

func (f *FailoverMethod) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var val string
	if err := unmarshal(&val); err != nil {
		return err
	} else {
		*f = failoverToID[val]
	}
	return nil
}

func (f FailoverMethod) String() string {
	return failoverToString[f]
}

type Failover struct {
	Duration      *time.Duration `yaml:"duration"`
	AlwaysBalance bool           `yaml:"always_balance"`
	Method        FailoverMethod `yaml:"method"`
	LockHostAfter *time.Duration `yaml:"lock_host_after"`
}

type Config struct {
	Proxy           []Proxy  `yaml:"proxy"`
	Check           Check    `yaml:"check"`
	Zabbix          Zabbix   `yaml:"zabbix"`
	Failover        Failover `yaml:"failover"`
	SkipOnlineCheck bool     `yaml:"skip_online_check"`
}

type Env struct {
	ConfigPath string `env:"CONFIG_PATH" envDefault:"/config/config.yaml"`
	LogLevel   string `env:"LOG_LEVEL" envDefault:"info"`
}

func GetEnv() (Env, error) {
	var envVar Env
	err := env.Parse(&envVar)
	return envVar, err
}

func GetConfig() (*Config, error) {
	var config Config
	if envVar, err := GetEnv(); err != nil {
		return &config, err
	} else {
		if f, err := ioutil.ReadFile(envVar.ConfigPath); err != nil {
			return &config, err
		} else {
			err = yaml.Unmarshal(f, &config)
			return &config, err
		}
	}
}
