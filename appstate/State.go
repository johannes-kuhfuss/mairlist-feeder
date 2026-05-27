// Package appstate contains mutable runtime state shared between app components.
package appstate

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/robfig/cron/v3"
)

type Metrics struct {
	FileNumber         *prometheus.GaugeVec
	MairListPlaying    *prometheus.GaugeVec
	Connected          *prometheus.GaugeVec
	EventCounters      *prometheus.GaugeVec
	CrawlIntervals     *prometheus.GaugeVec
	RunResults         *prometheus.CounterVec
	RunDurations       *prometheus.HistogramVec
	FastEventDurations *prometheus.HistogramVec
	LongEventDurations *prometheus.HistogramVec
}

type RuntimeState struct {
	Mu                    sync.Mutex
	Router                *gin.Engine
	BgJobs                *cron.Cron
	ListenAddr            string
	StartDate             time.Time
	CrawlRunNumber        int
	LastCrawlDate         time.Time
	LastExportRunDate     time.Time
	LastExportedFileDate  time.Time
	LastExportFileName    string
	CrawlRunning          bool
	ExportRunning         bool
	CleanRunning          bool
	LastCleanDate         time.Time
	FilesCleaned          int
	CrawlJobId            cron.EntryID
	ExportJobId           cron.EntryID
	CleanJobId            cron.EntryID
	EventJobId            cron.EntryID
	CalCmsJobId           cron.EntryID
	LastCalCmsState       string
	LastCalCmsRefreshDate time.Time
	LastCalCmsRefreshErr  string
	LastMairListCommState string
	MairListPlaying       bool
}

type AppState struct {
	Metrics Metrics
	Runtime RuntimeState
}

func New() *AppState {
	return &AppState{
		Runtime: RuntimeState{
			LastCalCmsState:       "N/A",
			LastMairListCommState: "N/A",
			LastCrawlDate:         time.Now(),
		},
	}
}

func (m *Metrics) SetFileNumber(kind string, value float64) {
	if m.FileNumber != nil {
		m.FileNumber.WithLabelValues(kind).Set(value)
	}
}

func (m *Metrics) SetConnected(subsystem string, value float64) {
	if m.Connected != nil {
		m.Connected.WithLabelValues(subsystem).Set(value)
	}
}

func (m *Metrics) SetEventCounter(kind string, value float64) {
	if m.EventCounters != nil {
		m.EventCounters.WithLabelValues(kind).Set(value)
	}
}

func (m *Metrics) SetMairListPlaying(name string, value float64) {
	if m.MairListPlaying != nil {
		m.MairListPlaying.WithLabelValues(name).Set(value)
	}
}

func (m *Metrics) SetCrawlInterval(kind string, value float64) {
	if m.CrawlIntervals != nil {
		m.CrawlIntervals.WithLabelValues(kind).Set(value)
	}
}

func (m *Metrics) ObserveFastEvent(name string, value float64) {
	if m.FastEventDurations != nil {
		m.FastEventDurations.WithLabelValues(name).Observe(value)
	}
}
