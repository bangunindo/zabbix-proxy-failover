package zabbix_proxy_failover

import (
	"context"
	"errors"
	"github.com/bangunindo/zabbix-proxy-failover/pkg/api"
	logger "github.com/sirupsen/logrus"
)

func ctxRequestTimeout(ctx context.Context) (context.Context, func()) {
	return context.WithTimeout(ctx, config.Check.Timeout)
}

func Failover(ctx context.Context) error {
	log := logger.WithField("module", "failover")
	z, err := getZabbix(ctx)
	if err != nil {
		return err
	}
	defer func(z api.ZabbixAPI) {
		log.Debug("logging out from zabbix")
		err := z.Logout()
		if err != nil {
			log.Debug("failed logging out from zabbix")
		}
	}(z)
	for {
		log.Debug("failover process started")
		ctxFailover, cancel := context.WithTimeout(ctx, config.Check.ExecutionDuration)
		err := doFailover(ctxFailover, z)
		if err != nil {
			log.WithError(err).Warnln("failover process failed")
		}
		log.Debug("failover process finished")
		cancel()
		ctxSchedule, cancel := context.WithTimeout(ctx, config.Check.Interval)
		select {
		case <-ctx.Done():
			cancel()
			return nil
		case <-ctxSchedule.Done():
		}
		cancel()
	}
}

func getZabbix(ctx context.Context) (api.ZabbixAPI, error) {
	log := logger.WithField("module", "failover.getZabbix")
	z := api.GetZabbixAPI(config.Zabbix.URL)
	ctxTimeout, cancel := ctxRequestTimeout(ctx)
	defer cancel()
	log.Debug("logging in to zabbix")
	err := z.LoginUserPassCtx(ctxTimeout, api.Login{
		User:     config.Zabbix.User,
		Password: config.Zabbix.Password,
	})
	log.Debug("logging in finished")
	return z, err
}

func doFailover(ctx context.Context, z api.ZabbixAPI) error {
	log := logger.WithField("module", "failover.process")
	ctxTimeout, cancel := ctxRequestTimeout(ctx)
	defer cancel()
	if err := z.Ping(ctxTimeout); err != nil {
		log.Warnln("failed connecting to zabbix api")
		return err
	}
	log.Debug("collecting proxy informations")
	var proxyIDs []int
	for _, proxy := range config.Proxy {
		proxyIDs = append(proxyIDs, proxy.ProxyID)
	}
	ctxTimeout, cancel = ctxRequestTimeout(ctx)
	defer cancel()
	proxies, err := z.GetProxyCtx(ctxTimeout, api.ReqProxy{ProxyIDs: proxyIDs})
	if err != nil {
		log.WithError(err).Warnln("failed collecting proxies")
		return err
	}
	if len(proxies) != len(proxyIDs) {
		log.Warnln("number of proxies returned doesn't match")
		return errors.New("proxy count mismatch")
	}
	log.Debug("analyzing any offline proxies")
	operation := Operation{}
	if err = operation.LoadProxy(ctx, z, proxies); err != nil {
		log.Warnln("failed loading proxy data")
		return errors.New("failed loading proxy data")
	}
	if err = operation.EvaluateStatus(ctx, z); err != nil {
		log.Warnln("failed evaluating proxy status")
		return errors.New("failed evaluating proxy status")
	}
	if err = operation.HostPlanning(); err != nil {
		log.Warnln("failed planning proxy hosts")
		return errors.New("failed planning proxy hosts")
	}
	if err = operation.Drain(ctx, z); err != nil {
		log.Warnln("failed draining hosts on proxy")
		return errors.New("failed draining hosts on proxy")
	}
	return nil
}
