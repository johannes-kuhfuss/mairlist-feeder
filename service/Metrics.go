package service

import (
	"time"

	"github.com/johannes-kuhfuss/mairlist-feeder/config"
)

func recordRunMetrics(cfg *config.AppConfig, serviceName string, start time.Time, err error) {
	if cfg == nil {
		return
	}
	result := "success"
	if err != nil {
		result = "failure"
	}
	if cfg.Metrics.RunResults != nil {
		cfg.Metrics.RunResults.WithLabelValues(serviceName, result).Inc()
	}
	if cfg.Metrics.RunDurations != nil {
		cfg.Metrics.RunDurations.WithLabelValues(serviceName, result).Observe(time.Since(start).Seconds())
	}
}
