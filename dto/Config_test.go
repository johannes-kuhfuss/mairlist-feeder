// package dto defines the data structures used to exchange information
package dto

import (
	"testing"
	"time"

	"github.com/johannes-kuhfuss/mairlist-feeder/appstate"
	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/robfig/cron/v3"
	"github.com/stretchr/testify/assert"
)

var (
	testConfig config.AppConfig
	testState  *appstate.AppState
)

func TestGetConfigReturnsNoError(t *testing.T) {
	config.InitConfig("", &testConfig)
	testState = appstate.New()
	resp := GetConfig(&testConfig, testState)

	assert.NotNil(t, resp)

	assert.EqualValues(t, "release", resp.GinMode)
	assert.EqualValues(t, "localhost", resp.ServerHost)
}

func TestConvertDateNoDateReturnsNA(t *testing.T) {
	d := convertDate(time.Time{})
	assert.EqualValues(t, "N/A", d)
}

func TestConvertDateDateReturnsString(t *testing.T) {
	ti := time.Date(2024, 9, 17, 11, 12, 13, 0, time.UTC)
	d := convertDateInLocation(ti, time.FixedZone("CEST", 2*60*60))
	assert.EqualValues(t, "2024-09-17 13:12:13 +0200 CEST", d)
}

func TestGetNextJobDateNoJobReturnsNA(t *testing.T) {
	testState = appstate.New()
	j := getNextJobDate(testState.Runtime.BgJobs, 1)
	assert.EqualValues(t, "N/A", j)
}

func NoOp() {
	// needed for next test
}

func TestGetNextJobDateJobReturnsDate(t *testing.T) {
	testState = appstate.New()
	testState.Runtime.BgJobs = cron.New()
	id, _ := testState.Runtime.BgJobs.AddFunc("@every 5m", NoOp)
	j := getNextJobDate(testState.Runtime.BgJobs, id)
	assert.EqualValues(t, "0001-01-01 00:00:00 +0000 UTC", j)
}

func TestGetStreamMappingsReturnsMappings(t *testing.T) {
	m := map[string]int{"A": 1}
	ma := getStreamMappings(m)
	assert.EqualValues(t, "A -> 1; ", ma)
}
