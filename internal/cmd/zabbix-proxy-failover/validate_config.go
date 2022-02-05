package zabbix_proxy_failover

import (
	"errors"
	"fmt"
	iconfig "github.com/bangunindo/zabbix-proxy-failover/internal/config"
	logger "github.com/sirupsen/logrus"
	"time"
)

var config *iconfig.Config

func ValidateConfig() error {
	log := logger.WithField("module", "validate.config")
	var err error
	log.Debug("getting config")
	config, err = iconfig.GetConfig()
	if err != nil {
		return err
	}
	log.Debug("checking check.interval config ", config.Check.Interval)
	if config.Check.Interval == 0 {
		config.Check.Interval = 5 * time.Minute
	}
	log.Debug("checking check.timeout config ", config.Check.Timeout)
	if config.Check.Timeout == 0 {
		config.Check.Timeout = 5 * time.Second
	}
	log.Debug("checking check.execution_duration config ", config.Check.ExecutionDuration)
	if config.Check.ExecutionDuration == 0 {
		config.Check.ExecutionDuration = 5 * time.Minute
	}
	log.Debug("checking failover.duration config ", config.Failover.Duration)
	if config.Failover.Duration == nil {
		if config.Failover.Method == iconfig.TRIGGER {
			*config.Failover.Duration = 0
		} else if config.Failover.Method == iconfig.LASTSEEN {
			*config.Failover.Duration = 5 * time.Minute
		}
	}
	log.Debug("checking zabbix config")
	errorHappened := false
	if config.Zabbix.URL == "" {
		errorHappened = true
		log.WithField("location", "zabbix.url").
			Errorln("field cannot be empty")
	}
	if config.Zabbix.User == "" {
		errorHappened = true
		log.WithField("location", "zabbix.user").
			Errorln("field cannot be empty")
	}
	if config.Zabbix.Password == "" {
		errorHappened = true
		log.WithField("location", "zabbix.password").
			Errorln("field cannot be empty")
	}
	log.Debug("checking proxy config")
	if len(config.Proxy) == 0 {
		errorHappened = true
		log.WithField("location", "proxy").
			Errorln("field cannot be empty")
	}
	proxyExists := map[int]bool{}
	for i, proxy := range config.Proxy {
		proxyLog := log.WithField("location", fmt.Sprintf("proxy[%d]", i))
		if proxy.ProxyID == 0 {
			errorHappened = true
			proxyLog.Errorln("proxy_id cannot be empty")
		} else if proxyExists[proxy.ProxyID] {
			errorHappened = true
			proxyLog.Errorln("duplicate proxy_id", proxy.ProxyID)
		} else {
			proxyExists[proxy.ProxyID] = true
		}
		if config.Failover.Method == iconfig.TRIGGER && proxy.TriggerID == nil {
			errorHappened = true
			proxyLog.Errorln("trigger_id has to be provided for trigger failover method")
		}
		if len(proxy.Tag) == 0 {
			proxyLog.Warnln("no tag defined, failover will always choose this proxy")
		}
		if proxy.Weight == 0 {
			proxyLog.Warnln("zero weight, failover won't choose this proxy")
		}
		for j, tag := range proxy.Tag {
			tagLog := log.WithField("location", fmt.Sprintf("proxy[%d].tag[%d]", i, j))
			if tag.Name == "" {
				tagLog.Warnln("empty tag name")
			}
			if tag.Value == "" {
				tagLog.Warnln("empty tag value")
			}
		}
	}
	if errorHappened {
		return errors.New("failed validating config")
	}
	return nil
}
