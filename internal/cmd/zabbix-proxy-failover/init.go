package zabbix_proxy_failover

import (
	iconfig "github.com/bangunindo/zabbix-proxy-failover/internal/config"
	log "github.com/sirupsen/logrus"
	"os"
)

func init() {
	log.SetOutput(os.Stdout)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02T15:04:05.0000",
	})
	if env, err := iconfig.GetEnv(); err != nil {
		log.Fatalln(err)
	} else {
		if level, err := log.ParseLevel(env.LogLevel); err != nil {
			log.SetLevel(log.InfoLevel)
		} else {
			//switch level {
			//case log.DebugLevel:
			//	log.SetReportCaller(true)
			//}
			log.SetLevel(level)
		}
	}
}
