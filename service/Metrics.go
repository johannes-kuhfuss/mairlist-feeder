package service

import (
	"time"

	"github.com/johannes-kuhfuss/mairlist-feeder/appstate"
)

func recordRunMetrics(state *appstate.AppState, serviceName string, start time.Time, err error) {
	if state == nil {
		return
	}
	result := "success"
	if err != nil {
		result = "failure"
	}
	if state.Metrics.RunResults != nil {
		state.Metrics.RunResults.WithLabelValues(serviceName, result).Inc()
	}
	if state.Metrics.RunDurations != nil {
		state.Metrics.RunDurations.WithLabelValues(serviceName, result).Observe(time.Since(start).Seconds())
	}
}
