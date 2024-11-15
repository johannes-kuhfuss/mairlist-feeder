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
		Name:      "mairlistfeeder_file_count",
		Help:      "Number of files managed by mAirList-Feeder",
	}, []string{
		"fileCountType",
	})
	cfg.Metrics.MairListPlaying = *prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "Coloradio",
		Subsystem: "mAirListFeeder",
		Name:      "mairlistfeeder_playstatus",
		Help:      "Status if mAirList is currently playing",
	}, []string{
		"mairlistname",
	})
	cfg.Metrics.Connected = *prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "Coloradio",
		Subsystem: "mAirListFeeder",
		Name:      "mairlistfeeder_subsystem_connection",
		Help:      "Status if mAirList-Feeder is connected to its subsystems",
	}, []string{
		"subsystemname",
	})
	cfg.Metrics.EventCounters = *prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "Coloradio",
		Subsystem: "mAirListFeeder",
		Name:      "mairlistfeeder_events_counters",
		Help:      "How many events have essence associated vs. missing vs. multiple essence",
	}, []string{
		"typename",
	})
	prometheus.MustRegister(cfg.Metrics.FileNumber)
	prometheus.MustRegister(cfg.Metrics.MairListPlaying)
	prometheus.MustRegister(cfg.Metrics.Connected)
	prometheus.MustRegister(cfg.Metrics.EventCounters)
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
}
