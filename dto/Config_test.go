// package dto defines the data structures used to exchange information
package dto

import (
	"testing"
	"time"

	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/robfig/cron/v3"
	"github.com/stretchr/testify/assert"
)

var (
	testConfig config.AppConfig
)

func Test_GetConfig_Returns_NoError(t *testing.T) {
	config.InitConfig("", &testConfig)
	resp := GetConfig(&testConfig)

	assert.NotNil(t, resp)

	assert.EqualValues(t, "release", resp.GinMode)
	assert.EqualValues(t, "localhost", resp.ServerHost)
}

func Test_convertDate_NoDate_Returns_NA(t *testing.T) {
	d := convertDate(time.Time{})
	assert.EqualValues(t, "N/A", d)
}

func Test_convertDate_Date_Returns_String(t *testing.T) {
	ti := time.Date(2024, 9, 17, 11, 12, 13, 0, time.UTC)
	d := convertDate(ti)
	assert.EqualValues(t, "2024-09-17 13:12:13 +0200 CEST", d)
}

func Test_getNextJobDate_NoJob_Returns_NA(t *testing.T) {
	config.InitConfig("", &testConfig)
	j := getNextJobDate(&testConfig, 1)
	assert.EqualValues(t, "N/A", j)
}

func NoOp() {
}

func Test_getNextJobDate_Job_Returns_Date(t *testing.T) {
	config.InitConfig("", &testConfig)
	testConfig.RunTime.BgJobs = cron.New()
	id, _ := testConfig.RunTime.BgJobs.AddFunc("@every 5m", NoOp)
	j := getNextJobDate(&testConfig, int(id))
	assert.EqualValues(t, "0001-01-01 00:00:00 +0000 UTC", j)
}

func Test_getStreamMappings_Returns_Mappings(t *testing.T) {
	m := map[string]int{"A": 1}
	ma := getStreamMappings(m)
	assert.EqualValues(t, "A -> 1; ", ma)
}
