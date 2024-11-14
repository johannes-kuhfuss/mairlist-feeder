// package app ties together all bits and pieces to start the program
package app

import (
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
	prometheus.MustRegister(cfg.Metrics.FileNumber)
	prometheus.MustRegister(cfg.Metrics.MairListPlaying)
}

func updateMetrics() {
	for {
		cfg.Metrics.FileNumber.WithLabelValues("total").Set(float64(cfg.RunTime.FilesInList))
		cfg.Metrics.FileNumber.WithLabelValues("audio").Set(float64(cfg.RunTime.AudioFilesInList))
		cfg.Metrics.FileNumber.WithLabelValues("stream").Set(float64(cfg.RunTime.StreamFilesInList))
		if cfg.RunTime.MairListPlaying {
			cfg.Metrics.MairListPlaying.WithLabelValues(cfg.Export.MairListUrl).Set(1)
		} else {
			cfg.Metrics.MairListPlaying.WithLabelValues(cfg.Export.MairListUrl).Set(0)
		}
		time.Sleep(3 * time.Second)
	}
}
