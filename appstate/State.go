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
	CrawlJobID            cron.EntryID
	ExportJobID           cron.EntryID
	CleanJobID            cron.EntryID
	EventJobID            cron.EntryID
	CalCmsJobID           cron.EntryID
	LastCalCmsState       string
	LastCalCmsRefreshDate time.Time
	LastCalCmsRefreshErr  string
	LastMairListCommState string
	MairListPlaying       bool
}

// RuntimeSnapshot is an immutable copy of runtime values safe for concurrent readers.
type RuntimeSnapshot struct {
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
	CrawlJobID            cron.EntryID
	ExportJobID           cron.EntryID
	CleanJobID            cron.EntryID
	EventJobID            cron.EntryID
	CalCmsJobID           cron.EntryID
	LastCalCmsState       string
	LastCalCmsRefreshDate time.Time
	LastCalCmsRefreshErr  string
	LastMairListCommState string
	MairListPlaying       bool
}

func (r *RuntimeState) Update(update func(*RuntimeState)) {
	r.Mu.Lock()
	defer r.Mu.Unlock()
	update(r)
}

func (r *RuntimeState) Snapshot() RuntimeSnapshot {
	r.Mu.Lock()
	defer r.Mu.Unlock()
	return RuntimeSnapshot{
		Router:                r.Router,
		BgJobs:                r.BgJobs,
		ListenAddr:            r.ListenAddr,
		StartDate:             r.StartDate,
		CrawlRunNumber:        r.CrawlRunNumber,
		LastCrawlDate:         r.LastCrawlDate,
		LastExportRunDate:     r.LastExportRunDate,
		LastExportedFileDate:  r.LastExportedFileDate,
		LastExportFileName:    r.LastExportFileName,
		CrawlRunning:          r.CrawlRunning,
		ExportRunning:         r.ExportRunning,
		CleanRunning:          r.CleanRunning,
		LastCleanDate:         r.LastCleanDate,
		FilesCleaned:          r.FilesCleaned,
		CrawlJobID:            r.CrawlJobID,
		ExportJobID:           r.ExportJobID,
		CleanJobID:            r.CleanJobID,
		EventJobID:            r.EventJobID,
		CalCmsJobID:           r.CalCmsJobID,
		LastCalCmsState:       r.LastCalCmsState,
		LastCalCmsRefreshDate: r.LastCalCmsRefreshDate,
		LastCalCmsRefreshErr:  r.LastCalCmsRefreshErr,
		LastMairListCommState: r.LastMairListCommState,
		MairListPlaying:       r.MairListPlaying,
	}
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
