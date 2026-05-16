// package app ties together all bits and pieces to start the program
package app

import (
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// initMetrics sets up the Prometheus metrics
func initMetrics() {
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
				30,
				60,
				120,
				180,
				240,
				300,
				360,
				420,
				480,
				540,
				600,
				900,
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

func updateMetrics() {
	for {
		doUpdate()
		time.Sleep(3 * time.Second)
	}
}

func doUpdate() {
	cfg.RunTime.Mu.Lock()
	defer cfg.RunTime.Mu.Unlock()
	cfg.Metrics.FileNumber.WithLabelValues("total").Set(float64(cfg.RunTime.FilesInList))
	cfg.Metrics.FileNumber.WithLabelValues("audio").Set(float64(cfg.RunTime.AudioFilesInList))
	cfg.Metrics.FileNumber.WithLabelValues("stream").Set(float64(cfg.RunTime.StreamFilesInList))
	if cfg.RunTime.MairListPlaying {
		cfg.Metrics.MairListPlaying.WithLabelValues(cfg.Export.MairListUrl).Set(1)
	} else {
		cfg.Metrics.MairListPlaying.WithLabelValues(cfg.Export.MairListUrl).Set(0)
	}
	if strings.Contains(cfg.RunTime.LastCalCmsState, "Succeeded") {
		cfg.Metrics.Connected.WithLabelValues("calCMS").Set(1)
	} else {
		cfg.Metrics.Connected.WithLabelValues("calCMS").Set(0)
	}
	if strings.Contains(cfg.RunTime.LastMairListCommState, "Succeeded") {
		cfg.Metrics.Connected.WithLabelValues("mAirList").Set(1)
	} else {
		cfg.Metrics.Connected.WithLabelValues("mAirList").Set(0)
	}
	cfg.Metrics.EventCounters.WithLabelValues("present").Set(float64(cfg.RunTime.EventsPresent))
	cfg.Metrics.EventCounters.WithLabelValues("missing").Set(float64(cfg.RunTime.EventsMissing))
	cfg.Metrics.EventCounters.WithLabelValues("multiple").Set(float64(cfg.RunTime.EventsMultiple))
	cfg.Metrics.EventCounters.WithLabelValues("total").Set(float64(cfg.RunTime.EventsPresent + cfg.RunTime.EventsMissing + cfg.RunTime.EventsMultiple))
	// Durations
	cfg.Metrics.LongEventDurations.WithLabelValues("sincelastcrawl").Observe(cfg.RunTime.DurationSinceLastCrawl.Seconds())
	cfg.Metrics.FastEventDurations.WithLabelValues("lastcrawl").Observe(cfg.RunTime.LastCrawlDuration.Seconds())
	cfg.Metrics.FastEventDurations.WithLabelValues("lastextraction").Observe(cfg.RunTime.LastExtractDuration.Seconds())
	cfg.Metrics.FastEventDurations.WithLabelValues("lasthash").Observe(cfg.RunTime.LastHashDuration.Seconds())
	cfg.Metrics.FastEventDurations.WithLabelValues("lastcalcmsupdate").Observe(cfg.RunTime.LastCalCmsUpdateDuration.Seconds())
}
