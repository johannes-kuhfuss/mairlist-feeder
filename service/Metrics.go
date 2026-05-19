package service

import "github.com/johannes-kuhfuss/mairlist-feeder/config"

func recordRunResult(cfg *config.AppConfig, serviceName string, err error) {
	if cfg == nil || cfg.Metrics.RunResults == nil {
		return
	}
	result := "success"
	if err != nil {
		result = "failure"
	}
	cfg.Metrics.RunResults.WithLabelValues(serviceName, result).Inc()
}
