package metrics

import (
	"testing"

	"github.com/johannes-kuhfuss/mairlist-feeder/appstate"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitMetricsRegistersUsableCollectors(t *testing.T) {
	registry := prometheus.NewRegistry()
	state := appstate.New()

	InitMetrics(state, registry)
	state.Metrics.SetFileNumber("total", 2)
	state.Metrics.SetConnected("calCMS", 1)
	state.Metrics.SetEventCounter("present", 2)
	state.Metrics.SetMairListPlaying("studio", 1)
	state.Metrics.SetCrawlInterval("sincelastcrawl", 60)
	state.Metrics.ObserveFastEvent("crawl", 0.5)

	families, err := registry.Gather()
	require.NoError(t, err)
	names := make([]string, 0, len(families))
	for _, family := range families {
		names = append(names, family.GetName())
	}
	assert.Contains(t, names, "coloradio_mairlistfeeder_file_count")
	assert.Contains(t, names, "coloradio_mairlistfeeder_subsystem_connection")
	assert.Contains(t, names, "coloradio_mairlistfeeder_fast_event_duration_seconds")
}

func TestUnregisterMetricsRemovesCollectors(t *testing.T) {
	registry := prometheus.NewRegistry()
	state := appstate.New()
	InitMetrics(state, registry)

	UnregisterMetrics(state, registry)

	families, err := registry.Gather()
	require.NoError(t, err)
	assert.Empty(t, families)
}
