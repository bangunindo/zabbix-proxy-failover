package zabbix_proxy_failover

import (
	"context"
	"errors"
	"fmt"
	"github.com/bangunindo/zabbix-proxy-failover/pkg/api"
	logger "github.com/sirupsen/logrus"
)

func ValidateZabbix(ctx context.Context) error {
	log := logger.WithField("module", "validate.zabbix")
	log.Debug("getting zabbix client")
	client := api.GetZabbixAPI(config.Zabbix.URL)
	log.Info("logging in to zabbix")
	ctxTimeout, cancel := ctxRequestTimeout(ctx)
	defer cancel()
	err := client.LoginUserPassCtx(ctxTimeout, api.Login{
		User:     config.Zabbix.User,
		Password: config.Zabbix.Password,
	})
	if err != nil {
		return err
	}
	defer func(client api.ZabbixAPI) {
		log.Debug("logging out from zabbix")
		err := client.Logout()
		if err != nil {
			log.Debug("failed logging out")
		} else {
			log.Debug("logout success")
		}
	}(client)
	errorHappened := false
	log.Debug("test api server")
	ctxTimeout, cancel = ctxRequestTimeout(ctx)
	defer cancel()
	if err := client.Ping(ctx); err != nil {
		errorHappened = true
		log.WithError(err).Errorln("cannot connect to api server")
	}
	log.Debug("checking proxy existence")
outerLoop:
	for i, proxy := range config.Proxy {
		proxyLog := log.WithField("location", fmt.Sprintf("proxy[%d]", i))
		ctxTimeout, cancel = ctxRequestTimeout(ctx)
		log.Debug("checking proxyid ", proxy.ProxyID)
		proxyZbx, err := client.GetProxyCtx(ctxTimeout, api.ReqProxy{ProxyIDs: []int{proxy.ProxyID}})
		if err != nil || len(proxyZbx) == 0 {
			errorHappened = true
			proxyLog.WithError(err).Errorln("failed getting proxy data from zabbix")
		}
		cancel()
		select {
		case <-ctx.Done():
			log.Debug("context cancelled")
			break outerLoop
		default:
		}
	}
	if errorHappened {
		return errors.New("failed validating zabbix")
	}
	return nil
}
