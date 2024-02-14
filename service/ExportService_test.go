package service

import (
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/domain"
	"github.com/johannes-kuhfuss/mairlist-feeder/helper"
	"github.com/johannes-kuhfuss/mairlist-feeder/repositories"
	"github.com/stretchr/testify/assert"
)

var (
	cfg           config.AppConfig
	fileRepo      repositories.DefaultFileRepository
	exportService DefaultExportService
)

func setupTest(t *testing.T) func() {
	config.InitConfig(config.EnvFile, &cfg)
	fileRepo = repositories.NewFileRepository(&cfg)
	exportService = NewExportService(&cfg, &fileRepo)
	return func() {
	}
}

func Test_buildHttpRequest_EmptyUrl_ReturnsError(t *testing.T) {
	tearDown := setupTest(t)
	defer tearDown()
	cfg.Export.MairListUrl = ""

	req, err := exportService.buildHttpRequest("test")

	assert.Nil(t, req)
	assert.NotNil(t, err)
	assert.EqualValues(t, "url cannot be empty", err.Error())
}

func Test_buildHttpRequest_WithUrl_ReturnsRequest(t *testing.T) {
	tearDown := setupTest(t)
	defer tearDown()
	cfg.Export.MairListUrl = "http://localhost:9300/"
	cfg.Export.MairListUser = "test"
	cfg.Export.MairListPassword = "test"

	req, err := exportService.buildHttpRequest("test")

	assert.NotNil(t, req)
	assert.Nil(t, err)
	assert.EqualValues(t, "http://localhost:9300/execute", req.URL.String())
	assert.EqualValues(t, "application/x-www-form-urlencoded", req.Header.Get("Content-Type"))
	assert.EqualValues(t, "Basic dGVzdDp0ZXN0", req.Header.Get("Authorization"))
	b, _ := io.ReadAll(req.Body)
	assert.EqualValues(t, "command=PLAYLIST+1+APPEND+test", string(b))
}

func Test_getNextHour_returnsNextHour(t *testing.T) {
	next := (time.Now().Hour()) + 1
	test := getNextHour()
	assert.EqualValues(t, fmt.Sprintf("%02d", next), test)
}

func Test_checkTime_noEndTime(t *testing.T) {
	type checkData struct {
		ok   bool
		slot string
	}
	timesToCheck := map[float64]checkData{
		60.0:   {false, "Slot: N/A"},   // 1 min
		300.0:  {false, "Slot: N/A"},   // 5 min
		1680.0: {false, "Slot: N/A"},   // 28 min
		1740.0: {true, "Slot: 30min"},  // 29 min
		1800.0: {true, "Slot: 30min"},  // 30 min
		1860.0: {true, "Slot: 30min"},  // 31 min
		1920.0: {false, "Slot: N/A"},   // 32 min
		2580.0: {false, "Slot: N/A"},   // 43 min
		2640.0: {true, "Slot: 45min"},  // 44 min
		2700.0: {true, "Slot: 45min"},  // 45 min
		2760.0: {true, "Slot: 45min"},  // 46 min
		2820.0: {false, "Slot: N/A"},   // 47 min
		3480.0: {false, "Slot: N/A"},   // 58 min
		3540.0: {true, "Slot: 60min"},  // 59 min
		3600.0: {true, "Slot: 60min"},  // 60 min
		3660.0: {true, "Slot: 60min"},  // 61 min
		3720.0: {false, "Slot: N/A"},   // 62 min
		5280.0: {false, "Slot: N/A"},   // 88 min
		5340.0: {true, "Slot: 90min"},  // 89 min
		5400.0: {true, "Slot: 90min"},  // 90 min
		5460.0: {true, "Slot: 90min"},  // 91 min
		5520.0: {false, "Slot: N/A"},   // 92 min
		7080.0: {false, "Slot: N/A"},   // 118 min
		7140.0: {true, "Slot: 120min"}, // 119 min
		7200.0: {true, "Slot: 120min"}, // 120 min
		7260.0: {true, "Slot: 120min"}, // 121 min
		7320.0: {false, "Slot: N/A"},   // 122 min
	}
	for length, data := range timesToCheck {
		fi := domain.FileInfo{
			Duration: length,
		}
		ok, detail := checkTime(fi, 1.0, 1.0)
		detailData := strings.Split(detail, ",")
		assert.EqualValues(t, data.ok, ok)
		assert.EqualValues(t, data.slot, strings.TrimSpace(detailData[1]))
	}
}

func Test_checkTime_withEndTime(t *testing.T) {
	fi := domain.FileInfo{
		Duration:  3600,
		StartTime: helper.TimeFromHourAndMinute(14, 0),
		EndTime:   helper.TimeFromHourAndMinute(15, 0),
	}
	ok, detail := checkTime(fi, 1.0, 1.0)

	assert.EqualValues(t, ok, true)
	assert.EqualValues(t, "Rounded actual duration: 60 min, Slot: 60min, Delta to slot: 0, planned duration: 60, delta to planned duration: 0", detail)
}
