package zabbix_proxy_failover

import (
	"context"
	logger "github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
)

func Run() {
	log := logger.WithField("module", "main")
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		log.Debug("waiting for signal")
		select {
		case <-sigs:
			log.Info("received graceful shutdown, shutting down application")
			cancel()
		case <-ctx.Done():
			log.Debug("context cancelled")
		}
		log.Debug("signal processed")
	}()
	log.Info("starting config validation")
	if err := ValidateConfig(); err != nil {
		log.WithError(err).Errorln("failed validating configuration")
		return
	}
	if !config.SkipOnlineCheck {
		log.Info("starting zabbix validation")
		if err := ValidateZabbix(ctx); err != nil {
			log.WithError(err).Errorln("failed validating zabbix")
			return
		}
	}
	log.Info("starting failover process")
	if err := Failover(ctx); err != nil {
		log.WithError(err).Errorln("failover failed")
	}
	log.Info("application shut down")
}
