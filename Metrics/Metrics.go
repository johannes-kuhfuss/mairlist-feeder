// package app ties together all bits and pieces to start the program
package metrics

import (
	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/prometheus/client_golang/prometheus"
)

// InitMetrics sets up the Prometheus metrics
func InitMetrics(cfg *config.AppConfig) {
	cfg.Metrics.FileNumber = *prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "Coloradio",
		Subsystem: "mAirListFeeder",
		Name:      "file_count",
		Help:      "Number of files managed by mAirList-Feeder",
	}, []string{
		"fileCountType",
	})
	cfg.Metrics.MairListPlaying = *prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "Coloradio",
		Subsystem: "mAirListFeeder",
		Name:      "playstatus",
		Help:      "Status if mAirList is currently playing",
	}, []string{
		"mairlistname",
	})
	cfg.Metrics.Connected = *prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "Coloradio",
		Subsystem: "mAirListFeeder",
		Name:      "subsystem_connection",
		Help:      "Status if mAirList-Feeder is connected to its subsystems",
	}, []string{
		"subsystemname",
	})
	cfg.Metrics.EventCounters = *prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "Coloradio",
		Subsystem: "mAirListFeeder",
		Name:      "event_counters",
		Help:      "How many events have essence associated vs. missing vs. multiple essence",
	}, []string{
		"typename",
	})
	cfg.Metrics.FastEventDurations = *prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "Coloradio",
			Subsystem: "mAirListFeeder",
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
	)
	cfg.Metrics.LongEventDurations = *prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "Coloradio",
			Subsystem: "mAirListFeeder",
			Name:      "long_event_duration_seconds",
			Help:      "Duration of a long event in seconds",

			Buckets: []float64{
				30,  // 30s
				60,  // 1m
				120, // 2m
				180, // 3m
				240, // 4m
				300, // 5m
				360, // 6m
				420, // 7m
				480, // 8m
				540, // 9m
				600, // 10m
				900, // 15m
			},
		},
		[]string{
			"eventname",
		},
	)

	prometheus.MustRegister(cfg.Metrics.FileNumber)
	prometheus.MustRegister(cfg.Metrics.MairListPlaying)
	prometheus.MustRegister(cfg.Metrics.Connected)
	prometheus.MustRegister(cfg.Metrics.EventCounters)
	prometheus.MustRegister(cfg.Metrics.FastEventDurations)
	prometheus.MustRegister(cfg.Metrics.LongEventDurations)
}

func UnregisterMetrics(cfg *config.AppConfig) {
	prometheus.Unregister(cfg.Metrics.FileNumber)
	prometheus.Unregister(cfg.Metrics.MairListPlaying)
	prometheus.Unregister(cfg.Metrics.Connected)
	prometheus.Unregister(cfg.Metrics.EventCounters)
	prometheus.Unregister(cfg.Metrics.FastEventDurations)
	prometheus.Unregister(cfg.Metrics.LongEventDurations)
}
