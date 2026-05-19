// Package metrics defines and registers Prometheus metrics.
package metrics

import (
	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/prometheus/client_golang/prometheus"
)

// InitMetrics sets up the Prometheus metrics.
func InitMetrics(cfg *config.AppConfig, registry prometheus.Registerer) {
	registry = registererOrDefault(registry)

	cfg.Metrics.FileNumber = registerGaugeVec(registry, prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "coloradio",
		Subsystem: "mairlistfeeder",
		Name:      "file_count",
		Help:      "Number of files managed by mAirList-Feeder",
	}, []string{
		"fileCountType",
	}))
	cfg.Metrics.MairListPlaying = registerGaugeVec(registry, prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "coloradio",
		Subsystem: "mairlistfeeder",
		Name:      "playstatus",
		Help:      "Status if mAirList is currently playing",
	}, []string{
		"mairlistname",
	}))
	cfg.Metrics.Connected = registerGaugeVec(registry, prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "coloradio",
		Subsystem: "mairlistfeeder",
		Name:      "subsystem_connection",
		Help:      "Status if mAirList-Feeder is connected to its subsystems",
	}, []string{
		"subsystemname",
	}))
	cfg.Metrics.EventCounters = registerGaugeVec(registry, prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "coloradio",
		Subsystem: "mairlistfeeder",
		Name:      "event_counters",
		Help:      "How many events have essence associated vs. missing vs. multiple essence",
	}, []string{
		"typename",
	}))
	cfg.Metrics.CrawlIntervals = registerGaugeVec(registry, prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "coloradio",
		Subsystem: "mairlistfeeder",
		Name:      "crawl_interval_seconds",
		Help:      "Seconds elapsed between crawl runs",
	}, []string{
		"eventname",
	}))
	cfg.Metrics.RunResults = registerCounterVec(registry, prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "coloradio",
		Subsystem: "mairlistfeeder",
		Name:      "run_results_total",
		Help:      "Number of completed service runs by service and result",
	}, []string{
		"service",
		"result",
	}))
	cfg.Metrics.RunDurations = registerHistogramVec(registry, prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "coloradio",
			Subsystem: "mairlistfeeder",
			Name:      "run_duration_seconds",
			Help:      "Duration of completed service runs in seconds",
			Buckets: []float64{
				0.1, 0.5, 1, 2.5, 5, 10, 30, 60, 120, 300, 600, 900,
			},
		},
		[]string{
			"service",
			"result",
		},
	))
	cfg.Metrics.FastEventDurations = registerHistogramVec(registry, prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "coloradio",
			Subsystem: "mairlistfeeder",
			Name:      "fast_event_duration_seconds",
			Help:      "Duration of a fast event in seconds",
			Buckets: []float64{
				0.001, 0.005, 0.01, 0.025, 0.05,
				0.1, 0.25, 0.5, 1, 2.5, 5,
			},
		},
		[]string{
			"eventname",
		},
	))
}

// UnregisterMetrics removes the configured metrics from the given registry.
func UnregisterMetrics(cfg *config.AppConfig, registry prometheus.Registerer) {
	registry = registererOrDefault(registry)

	unregister(registry, cfg.Metrics.FileNumber)
	unregister(registry, cfg.Metrics.MairListPlaying)
	unregister(registry, cfg.Metrics.Connected)
	unregister(registry, cfg.Metrics.EventCounters)
	unregister(registry, cfg.Metrics.CrawlIntervals)
	unregister(registry, cfg.Metrics.RunResults)
	unregister(registry, cfg.Metrics.RunDurations)
	unregister(registry, cfg.Metrics.FastEventDurations)
}

func registerGaugeVec(registry prometheus.Registerer, metric *prometheus.GaugeVec) *prometheus.GaugeVec {
	if err := registry.Register(metric); err != nil {
		if alreadyRegistered, ok := err.(prometheus.AlreadyRegisteredError); ok {
			if existing, ok := alreadyRegistered.ExistingCollector.(*prometheus.GaugeVec); ok {
				return existing
			}
		}
		panic(err)
	}
	return metric
}

func registerCounterVec(registry prometheus.Registerer, metric *prometheus.CounterVec) *prometheus.CounterVec {
	if err := registry.Register(metric); err != nil {
		if alreadyRegistered, ok := err.(prometheus.AlreadyRegisteredError); ok {
			if existing, ok := alreadyRegistered.ExistingCollector.(*prometheus.CounterVec); ok {
				return existing
			}
		}
		panic(err)
	}
	return metric
}

func registerHistogramVec(registry prometheus.Registerer, metric *prometheus.HistogramVec) *prometheus.HistogramVec {
	if err := registry.Register(metric); err != nil {
		if alreadyRegistered, ok := err.(prometheus.AlreadyRegisteredError); ok {
			if existing, ok := alreadyRegistered.ExistingCollector.(*prometheus.HistogramVec); ok {
				return existing
			}
		}
		panic(err)
	}
	return metric
}

func unregister(registry prometheus.Registerer, metric prometheus.Collector) {
	if metric != nil {
		registry.Unregister(metric)
	}
}

func registererOrDefault(registry prometheus.Registerer) prometheus.Registerer {
	if registry != nil {
		return registry
	}
	return prometheus.DefaultRegisterer
}
