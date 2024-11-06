package service

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
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

func setupTestEx() func() {
	config.InitConfig(config.EnvFile, &cfg)
	fileRepo = repositories.NewFileRepository(&cfg)
	exportService = NewExportService(&cfg, &fileRepo)
	return func() {
		fileRepo.DeleteAllData()
		fileExportList.Files = nil
	}
}

func TestBuildHttpRequestEmptyUrlReturnsError(t *testing.T) {
	tearDown := setupTestEx()
	defer tearDown()
	cfg.Export.MairListUrl = ""

	req, err := exportService.buildHttpRequest("test")

	assert.Nil(t, req)
	assert.NotNil(t, err)
	assert.EqualValues(t, "url cannot be empty", err.Error())
}

func TestBuildHttpRequestWithUrlReturnsRequest(t *testing.T) {
	tearDown := setupTestEx()
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

func TestGetNextHourreturnsNextHour(t *testing.T) {
	next := (time.Now().Hour()) + 1
	test := getNextHour()
	assert.EqualValues(t, fmt.Sprintf("%02d", next), test)
}

func TestCheckTimeNoEndTime(t *testing.T) {
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
		ok, _, detail := checkTime(fi, 1.0, 1.0)
		detailData := strings.Split(detail, ",")
		assert.EqualValues(t, data.ok, ok)
		assert.EqualValues(t, data.slot, strings.TrimSpace(detailData[1]))
	}
}

func TestCheckTimeWithEndTime(t *testing.T) {
	fi := domain.FileInfo{
		Duration:  3600,
		StartTime: helper.TimeFromHourAndMinute(14, 0),
		EndTime:   helper.TimeFromHourAndMinute(15, 0),
	}
	ok, _, detail := checkTime(fi, 1.0, 1.0)

	assert.EqualValues(t, ok, true)
	assert.EqualValues(t, "Rounded actual duration: 60 min, Slot: 60min, Delta to slot: 0, planned duration: 60, delta to planned duration: 0", detail)
}

func TestSetStartTimeOneTimeValue(t *testing.T) {
	var st time.Time

	st = setStartTime(st, "14:00")
	tt, _ := time.Parse("15:04", "14:00")

	assert.EqualValues(t, st, tt)
}

func TestSetStartTimeTwoTimeValuesReturnsEarlier1(t *testing.T) {
	var st time.Time

	st = setStartTime(st, "14:00")
	st = setStartTime(st, "14:30")
	tt, _ := time.Parse("15:04", "14:00")

	assert.EqualValues(t, st, tt)
}

func TestSetStartTimeTwoTimeValuesReturnsEarlier2(t *testing.T) {
	var st time.Time

	st = setStartTime(st, "14:30")
	st = setStartTime(st, "14:00")
	tt, _ := time.Parse("15:04", "14:00")

	assert.EqualValues(t, st, tt)
}

func TestAppendPlaylistUrlNotFoundReturnsError(t *testing.T) {
	tearDown := setupTestEx()
	defer tearDown()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("nok"))
	}))
	defer srv.Close()
	exportService.Cfg.Export.MairListUrl = srv.URL
	err := exportService.AppendPlaylist("yfile.txt")
	assert.NotNil(t, err)
	assert.EqualValues(t, "url not found", err.Error())
}

func TestAppendPlaylistMairListErrorReturnsError(t *testing.T) {
	tearDown := setupTestEx()
	defer tearDown()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("nok"))
	}))
	defer srv.Close()
	exportService.Cfg.Export.MairListUrl = srv.URL
	err := exportService.AppendPlaylist("yfile.txt")
	assert.NotNil(t, err)
	assert.EqualValues(t, "nok", err.Error())
}

func TestAppendPlaylistMairListOkReturnsNil(t *testing.T) {
	tearDown := setupTestEx()
	defer tearDown()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer srv.Close()
	exportService.Cfg.Export.MairListUrl = srv.URL
	err := exportService.AppendPlaylist("yfile.txt")
	assert.Nil(t, err)
}

func TestSetExportPathTestReturnsTest(t *testing.T) {
	tearDown := setupTestEx()
	defer tearDown()
	exportService.Cfg.Misc.TestCrawl = true
	s, _ := exportService.setExportPath("13")
	assert.NotNil(t, s)
	assert.EqualValues(t, "C:\\TEMP\\Test_13.tpi", s)
}

func TestSetExportPathRegularReturnsPath(t *testing.T) {
	tearDown := setupTestEx()
	defer tearDown()
	hour := "13"
	s, _ := exportService.setExportPath(hour)
	assert.NotNil(t, s)
	file := time.Now().Format("2006-01-02") + "-" + hour + ".tpi"
	tp := path.Join(exportService.Cfg.Export.ExportFolder, file)

	assert.EqualValues(t, strings.Replace(tp, "/", "\\", -1), s)
}

func TestCheckTimeAndLenghthOneFile(t *testing.T) {
	var files domain.FileList
	tearDown := setupTestEx()
	defer tearDown()
	fi := domain.FileInfo{
		Duration:  3600,
		StartTime: helper.TimeFromHourAndMinute(14, 0),
		EndTime:   helper.TimeFromHourAndMinute(15, 0),
	}
	files = append(files, fi)
	assert.EqualValues(t, 0, len(fileExportList.Files))
	exportService.checkTimeAndLenghth(&files)
	assert.EqualValues(t, 1, len(fileExportList.Files))
}

func TestCheckTimeAndLenghthOneFileSame(t *testing.T) {
	var files domain.FileList
	tearDown := setupTestEx()
	defer tearDown()
	fi := domain.FileInfo{
		Path:      "A",
		Duration:  3600,
		StartTime: helper.TimeFromHourAndMinute(14, 0),
		EndTime:   helper.TimeFromHourAndMinute(15, 0),
	}
	files = append(files, fi)
	assert.EqualValues(t, 0, len(fileExportList.Files))
	exportService.checkTimeAndLenghth(&files)
	assert.EqualValues(t, 1, len(fileExportList.Files))
	exportService.checkTimeAndLenghth(&files)
	assert.EqualValues(t, 1, len(fileExportList.Files))
	assert.EqualValues(t, "A", fileExportList.Files["14:00"].Path)
}

func TestCheckTimeAndLenghthOneFileNewer(t *testing.T) {
	var files domain.FileList
	tearDown := setupTestEx()
	defer tearDown()
	fi1 := domain.FileInfo{
		Path:      "A",
		Duration:  3600,
		StartTime: helper.TimeFromHourAndMinute(14, 0),
		EndTime:   helper.TimeFromHourAndMinute(15, 0),
		ModTime:   time.Now(),
	}
	fi2 := domain.FileInfo{
		Path:      "A2",
		Duration:  3600,
		StartTime: helper.TimeFromHourAndMinute(14, 0),
		EndTime:   helper.TimeFromHourAndMinute(15, 0),
		ModTime:   time.Now().AddDate(0, 0, -1),
	}
	files = append(files, fi1)
	assert.EqualValues(t, 0, len(fileExportList.Files))
	exportService.checkTimeAndLenghth(&files)
	assert.EqualValues(t, 1, len(fileExportList.Files))
	files = append(files, fi2)
	exportService.checkTimeAndLenghth(&files)
	assert.EqualValues(t, 1, len(fileExportList.Files))
	assert.EqualValues(t, "A", fileExportList.Files["14:00"].Path)
}

func TestExportToPlayoutNoFilesNoExport(t *testing.T) {
	tearDown := setupTestEx()
	defer tearDown()
	file, err := exportService.ExportToPlayout("13")
	assert.EqualValues(t, "", file)
	assert.Nil(t, err)
}

func TestExportToPlayoutOneFilesExport(t *testing.T) {
	var fileLines []string
	tearDown := setupTestEx()
	defer tearDown()
	fi := domain.FileInfo{
		Path:       "A",
		Duration:   3600,
		StartTime:  helper.TimeFromHourAndMinute(13, 0),
		EndTime:    helper.TimeFromHourAndMinute(14, 0),
		SlotLength: 60.0,
	}
	fileExportList.Files["13:00"] = fi
	file, err := exportService.ExportToPlayout("13")
	assert.Nil(t, err)
	readFile, _ := os.Open(file)
	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)
	for fileScanner.Scan() {
		fileLines = append(fileLines, fileScanner.Text())
	}
	readFile.Close()
	assert.EqualValues(t, "13:00:00\tH\tF\tA", fileLines[1])
	assert.EqualValues(t, "14:00:00\tH\tD\tEnd of block", fileLines[2])
	assert.EqualValues(t, "\t\tR\tEnd of auto-generated playlist", fileLines[3])
	time.Sleep(1 * time.Second)
	os.Remove(file)
	assert.EqualValues(t, file, exportService.Cfg.RunTime.LastExportFileName)
	assert.GreaterOrEqual(t, time.Now(), exportService.Cfg.RunTime.LastExportedFileDate)
}
