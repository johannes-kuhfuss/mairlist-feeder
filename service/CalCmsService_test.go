package service

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/domain"
	"github.com/johannes-kuhfuss/mairlist-feeder/helper"
	"github.com/johannes-kuhfuss/mairlist-feeder/repositories"
	"github.com/stretchr/testify/assert"
)

var (
	cfgCal        config.AppConfig
	fileRepoCal   repositories.DefaultFileRepository
	calCmsService DefaultCalCmsService
)

func setupTestCal() func() {
	config.InitConfig(config.EnvFile, &cfgCal)
	fileRepoCal = repositories.NewFileRepository(&cfgCal)
	calCmsService = NewCalCmsService(&cfgCal, &fileRepoCal)
	return func() {
		fileRepoCal.DeleteAllData()
	}
}

func Test_convertToEntry_TimeError1_ReturnsError(t *testing.T) {
	teardown := setupTestCal()
	defer teardown()
	ev := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartDatetime: "AA:BB",
		EndDatetime:   "CC:DD",
		EventID:       1,
	}
	_, err := calCmsService.convertToEntry(ev)
	assert.NotNil(t, err)
	assert.EqualValues(t, "parsing time \"AA:BB\" as \"2006-01-02T15:04:05\": cannot parse \"AA:BB\" as \"2006\"", err.Error())
}

func Test_convertToEntry_TimeError2_ReturnsError(t *testing.T) {
	teardown := setupTestCal()
	defer teardown()
	ev := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartDatetime: "11:00",
		EndDatetime:   "CC:DD",
		EventID:       1,
	}
	_, err := calCmsService.convertToEntry(ev)
	assert.NotNil(t, err)
	assert.EqualValues(t, "parsing time \"11:00\" as \"2006-01-02T15:04:05\": cannot parse \"11:00\" as \"2006\"", err.Error())
}

func Test_convertToEntry_NoError_ReturnsEntry(t *testing.T) {
	teardown := setupTestCal()
	defer teardown()
	ev := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartDatetime: "2024-04-01T11:00:00",
		EndDatetime:   "2024-04-01T12:00:00",
		EventID:       12345,
	}
	res, err := calCmsService.convertToEntry(ev)
	assert.Nil(t, err)
	t1, _ := time.ParseInLocation("2006-01-02T15:04:05", "2024-04-01T11:00:00", time.Local)
	t2, _ := time.ParseInLocation("2006-01-02T15:04:05", "2024-04-01T12:00:00", time.Local)
	assert.EqualValues(t, t1, res.StartTime)
	assert.EqualValues(t, t2, res.EndTime)
	assert.EqualValues(t, "Test", res.Title)
	assert.EqualValues(t, 12345, res.EventId)
}

func Test_GetCalCmsDataForId_Empty_ReturnsEmptyList(t *testing.T) {
	teardown := setupTestCal()
	defer teardown()
	res, err := calCmsService.GetCalCmsDataForId(1)
	assert.Nil(t, err)
	assert.EqualValues(t, 0, len(res))
}

func Test_GetCalCmsDataForId_WrongData_ReturnsError(t *testing.T) {
	var events []domain.CalCmsEvent
	teardown := setupTestCal()
	defer teardown()

	event := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartDatetime: "2024-04-01T11:00:00",
		EndDatetime:   "CC:DD",
		EventID:       1,
	}
	events = append(events, event)
	data := domain.CalCmsPgmData{
		Events: events,
	}
	calCmsService.insertData(data)

	res, err := calCmsService.GetCalCmsDataForId(1)
	assert.NotNil(t, err)
	assert.EqualValues(t, 0, len(res))
	assert.EqualValues(t, "parsing time \"CC:DD\" as \"2006-01-02T15:04:05\": cannot parse \"CC:DD\" as \"2006\"", err.Error())
}

func Test_GetCalCmsDataForId_WrongId_ReturnsEmpty(t *testing.T) {
	var events []domain.CalCmsEvent
	teardown := setupTestCal()
	defer teardown()

	event := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartDatetime: "2024-04-01T11:00:00",
		EndDatetime:   "2024-04-01T12:00:00",
		EventID:       2,
	}
	events = append(events, event)
	data := domain.CalCmsPgmData{
		Events: events,
	}
	calCmsService.insertData(data)

	res, err := calCmsService.GetCalCmsDataForId(1)
	assert.Nil(t, err)
	assert.EqualValues(t, 0, len(res))
}

func Test_GetCalCmsDataForId_OneElement_ReturnsData(t *testing.T) {
	var events []domain.CalCmsEvent
	teardown := setupTestCal()
	defer teardown()

	event := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartDatetime: "2024-04-01T11:00:00",
		EndDatetime:   "2024-04-01T12:00:00",
		EventID:       1,
	}
	events = append(events, event)
	data := domain.CalCmsPgmData{
		Events: events,
	}
	calCmsService.insertData(data)

	res, err := calCmsService.GetCalCmsDataForId(1)
	assert.Nil(t, err)
	assert.EqualValues(t, 1, len(res))
	assert.EqualValues(t, time.Duration(1*time.Hour), res[0].Duration)
}

func Test_GetCalCmsDataForId_TwoElements_ReturnsOne(t *testing.T) {
	var events []domain.CalCmsEvent
	teardown := setupTestCal()
	defer teardown()

	event1 := domain.CalCmsEvent{
		FullTitle:     "Test1",
		StartDatetime: "2024-04-01T11:00:00",
		EndDatetime:   "2024-04-01T12:00:00",
		EventID:       1,
	}
	event2 := domain.CalCmsEvent{
		FullTitle:     "Test2",
		StartDatetime: "2024-04-01T13:00:00",
		EndDatetime:   "2024-04-01T15:00:00",
		EventID:       2,
	}
	events = append(events, event1)
	events = append(events, event2)
	data := domain.CalCmsPgmData{
		Events: events,
	}
	calCmsService.insertData(data)

	res, err := calCmsService.GetCalCmsDataForId(1)
	assert.Nil(t, err)
	assert.EqualValues(t, 1, len(res))
	assert.EqualValues(t, time.Duration(1*time.Hour), res[0].Duration)
}

func Test_GetCalCmsDataForHour_Empty_ReturnsEmptyList(t *testing.T) {
	teardown := setupTestCal()
	defer teardown()
	res, err := calCmsService.GetCalCmsDataForHour("12:00")
	assert.Nil(t, err)
	assert.EqualValues(t, 0, len(res))
}

func Test_GetCalCmsDataForHour_OneElement_ReturnsData(t *testing.T) {
	var events []domain.CalCmsEvent
	teardown := setupTestCal()
	defer teardown()

	event := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartDatetime: "2024-04-01T11:00:00",
		StartTimeName: "11:00",
		EndDatetime:   "2024-04-01T12:00:00",
		EventID:       1,
	}
	events = append(events, event)
	data := domain.CalCmsPgmData{
		Events: events,
	}
	calCmsService.insertData(data)

	res, err := calCmsService.GetCalCmsDataForHour("11")
	assert.Nil(t, err)
	assert.EqualValues(t, 1, len(res))
	assert.EqualValues(t, time.Duration(1*time.Hour), res[0].Duration)
}

func Test_checkCalCmsData_WrongData_ReturnsError(t *testing.T) {
	var events []domain.CalCmsEvent
	teardown := setupTestCal()
	defer teardown()

	event := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartDatetime: "2024-04-01T11:00:00",
		EndDatetime:   "CC:DD",
		EventID:       1,
	}
	events = append(events, event)
	data := domain.CalCmsPgmData{
		Events: events,
	}
	calCmsService.insertData(data)
	fi := domain.FileInfo{
		Path:          "",
		ModTime:       time.Time{},
		Duration:      0,
		StartTime:     time.Time{},
		EndTime:       time.Time{},
		FromCalCMS:    false,
		InfoExtracted: false,
		ScanTime:      time.Time{},
		FolderDate:    "",
		RuleMatched:   "",
		EventId:       1,
		CalCmsTitle:   "",
	}

	_, err := calCmsService.checkCalCmsData(fi)

	assert.NotNil(t, err)
	assert.EqualValues(t, "parsing time \"CC:DD\" as \"2006-01-02T15:04:05\": cannot parse \"CC:DD\" as \"2006\"", err.Error())
}

func Test_checkCalCmsData_DifferingDates_ReturnsError(t *testing.T) {
	var events []domain.CalCmsEvent
	teardown := setupTestCal()
	defer teardown()

	event := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartDatetime: "2024-04-01T11:00:00",
		EndDatetime:   "2024-04-01T12:00:00",
		EventID:       1,
	}
	events = append(events, event)
	data := domain.CalCmsPgmData{
		Events: events,
	}
	calCmsService.insertData(data)
	fi := domain.FileInfo{
		Path:          "",
		ModTime:       time.Time{},
		Duration:      0,
		StartTime:     time.Time{},
		EndTime:       time.Time{},
		FromCalCMS:    false,
		InfoExtracted: false,
		ScanTime:      time.Time{},
		FolderDate:    "2024-09-17",
		RuleMatched:   "",
		EventId:       0,
		CalCmsTitle:   "",
	}

	_, err := calCmsService.checkCalCmsData(fi)

	assert.NotNil(t, err)
	assert.EqualValues(t, "file has different date from calCmsData", err.Error())
}

func Test_checkCalCmsData_NoMatchingOnSameDayData_ReturnsError(t *testing.T) {
	var events []domain.CalCmsEvent
	teardown := setupTestCal()
	defer teardown()

	event := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartDatetime: "2024-04-01T11:00:00",
		EndDatetime:   "2024-04-01T12:00:00",
		EventID:       1,
	}
	events = append(events, event)
	data := domain.CalCmsPgmData{
		Events: events,
	}
	calCmsService.insertData(data)
	fi := domain.FileInfo{
		Path:          "",
		ModTime:       time.Time{},
		Duration:      0,
		StartTime:     time.Time{},
		EndTime:       time.Time{},
		FromCalCMS:    false,
		InfoExtracted: false,
		ScanTime:      time.Time{},
		FolderDate:    "2024-09-17",
		RuleMatched:   "",
		EventId:       22,
		CalCmsTitle:   "",
	}

	calCmsService.Cfg.Misc.TestCrawl = true
	calCmsService.Cfg.Misc.TestDate = "2024/09/17"

	_, err := calCmsService.checkCalCmsData(fi)

	assert.NotNil(t, err)
	assert.EqualValues(t, "no such id in calCMS", err.Error())
}

func Test_checkCalCmsData_DoubleMatch_ReturnsError(t *testing.T) {
	var events []domain.CalCmsEvent
	teardown := setupTestCal()
	defer teardown()

	event1 := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartDatetime: "2024-04-01T11:00:00",
		EndDatetime:   "2024-04-01T12:00:00",
		EventID:       1,
	}
	event2 := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartDatetime: "2024-04-01T12:00:00",
		EndDatetime:   "2024-04-01T13:00:00",
		EventID:       1,
	}
	events = append(events, event1)
	events = append(events, event2)
	data := domain.CalCmsPgmData{
		Events: events,
	}
	calCmsService.insertData(data)
	fi := domain.FileInfo{
		Path:          "",
		ModTime:       time.Time{},
		Duration:      0,
		StartTime:     time.Time{},
		EndTime:       time.Time{},
		FromCalCMS:    false,
		InfoExtracted: false,
		ScanTime:      time.Time{},
		FolderDate:    "2024-09-17",
		RuleMatched:   "",
		EventId:       1,
		CalCmsTitle:   "",
	}

	calCmsService.Cfg.Misc.TestCrawl = true
	calCmsService.Cfg.Misc.TestDate = "2024/09/17"

	_, err := calCmsService.checkCalCmsData(fi)

	assert.NotNil(t, err)
	assert.EqualValues(t, "multiple matches in calCMS", err.Error())
}

func Test_checkCalCmsData_IsLive_ReturnsError(t *testing.T) {
	var events []domain.CalCmsEvent
	teardown := setupTestCal()
	defer teardown()

	event := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartDatetime: "2024-04-01T11:00:00",
		EndDatetime:   "2024-04-01T12:00:00",
		EventID:       1,
		Live:          1,
	}
	events = append(events, event)
	data := domain.CalCmsPgmData{
		Events: events,
	}
	calCmsService.insertData(data)
	fi := domain.FileInfo{
		Path:          "",
		ModTime:       time.Time{},
		Duration:      0,
		StartTime:     time.Time{},
		EndTime:       time.Time{},
		FromCalCMS:    false,
		InfoExtracted: false,
		ScanTime:      time.Time{},
		FolderDate:    "2024-09-17",
		RuleMatched:   "",
		EventId:       1,
		CalCmsTitle:   "",
	}

	calCmsService.Cfg.Misc.TestCrawl = true
	calCmsService.Cfg.Misc.TestDate = "2024/09/17"

	_, err := calCmsService.checkCalCmsData(fi)

	assert.NotNil(t, err)
	assert.EqualValues(t, "event is live in calCMS", err.Error())
}

func Test_checkCalCmsData_DataOk_ReturnsData(t *testing.T) {
	var events []domain.CalCmsEvent
	teardown := setupTestCal()
	defer teardown()

	event := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartDatetime: "2024-04-01T11:00:00",
		EndDatetime:   "2024-04-01T12:00:00",
		EventID:       1,
	}
	events = append(events, event)
	data := domain.CalCmsPgmData{
		Events: events,
	}
	calCmsService.insertData(data)
	fd := time.Date(2024, 4, 1, 0, 0, 0, 0, time.Local)
	fi := domain.FileInfo{
		Path:          "",
		ModTime:       time.Time{},
		Duration:      0,
		StartTime:     helper.TimeFromHourAndMinuteAndDate(11, 0, fd),
		EndTime:       time.Time{},
		FromCalCMS:    false,
		InfoExtracted: false,
		ScanTime:      time.Time{},
		FolderDate:    "2024-09-17",
		RuleMatched:   "",
		EventId:       1,
		CalCmsTitle:   "",
	}

	calCmsService.Cfg.Misc.TestCrawl = true
	calCmsService.Cfg.Misc.TestDate = "2024/09/17"

	res, err := calCmsService.checkCalCmsData(fi)

	assert.Nil(t, err)
	assert.EqualValues(t, event.FullTitle, res.Title)
	st1, _ := time.ParseInLocation("2006-01-02T15:04:05", event.StartDatetime, time.Local)
	st2, _ := time.ParseInLocation("2006-01-02T15:04:05", event.EndDatetime, time.Local)
	assert.EqualValues(t, fmt.Sprintf("%02d:%02d", st1.Hour(), st1.Minute()), res.StartTime.Format("15:04"))
	assert.EqualValues(t, fmt.Sprintf("%02d:%02d", st2.Hour(), st2.Minute()), res.EndTime.Format("15:04"))
}

func Test_EnrichFileInformation_NoFiles_Returns_Zero(t *testing.T) {
	teardown := setupTestCal()
	defer teardown()
	n := calCmsService.EnrichFileInformation()
	assert.EqualValues(t, 0, n.TotalCount)
}

func Test_EnrichFileInformation_OneFile_Returns_Enriched(t *testing.T) {
	var events []domain.CalCmsEvent
	teardown := setupTestCal()
	defer teardown()

	event := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartDatetime: "2024-04-01T11:00:00",
		EndDatetime:   "2024-04-01T12:00:00",
		EventID:       1234,
	}
	events = append(events, event)
	data := domain.CalCmsPgmData{
		Events: events,
	}
	calCmsService.insertData(data)
	fi := domain.FileInfo{
		Path:          "A",
		ModTime:       time.Time{},
		Duration:      0,
		StartTime:     time.Time{},
		EndTime:       time.Time{},
		FromCalCMS:    false,
		InfoExtracted: false,
		ScanTime:      time.Time{},
		FolderDate:    "2024-09-17",
		RuleMatched:   "",
		EventId:       1234,
		CalCmsTitle:   "",
		FileType:      "Audio",
	}
	fileRepoCal.Store(fi)

	calCmsService.Cfg.Misc.TestCrawl = true
	calCmsService.Cfg.Misc.TestDate = "2024/09/17"

	n := calCmsService.EnrichFileInformation()

	assert.EqualValues(t, 1, n.TotalCount)
	assert.EqualValues(t, 1, n.AudioCount)
	assert.EqualValues(t, 0, n.StreamCount)
}

func Test_EnrichFileInformation_OneStream_Returns_Enriched(t *testing.T) {
	var events []domain.CalCmsEvent
	teardown := setupTestCal()
	defer teardown()

	event := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartDatetime: "2024-04-01T11:00:00",
		EndDatetime:   "2024-04-01T12:00:00",
		EventID:       1234,
	}
	events = append(events, event)
	data := domain.CalCmsPgmData{
		Events: events,
	}
	calCmsService.insertData(data)
	fi := domain.FileInfo{
		Path:          "A",
		ModTime:       time.Time{},
		Duration:      0,
		StartTime:     time.Time{},
		EndTime:       time.Time{},
		FromCalCMS:    false,
		InfoExtracted: false,
		ScanTime:      time.Time{},
		FolderDate:    "2024-09-17",
		RuleMatched:   "",
		EventId:       1234,
		CalCmsTitle:   "",
		FileType:      "Stream",
		StreamId:      55,
	}
	fileRepoCal.Store(fi)

	calCmsService.Cfg.Misc.TestCrawl = true
	calCmsService.Cfg.Misc.TestDate = "2024/09/17"

	n := calCmsService.EnrichFileInformation()

	assert.EqualValues(t, 1, n.TotalCount)
	assert.EqualValues(t, 1, n.StreamCount)
	assert.EqualValues(t, 0, n.AudioCount)
}

func Test_getCalCmsData_WrongUrl_Returns_Error(t *testing.T) {
	teardown := setupTestCal()
	defer teardown()
	cfgCal.CalCms.CmsUrl = "§$%&/()"
	data, err := calCmsService.getCalCmsData()
	assert.Nil(t, data)
	assert.NotNil(t, err)
	assert.EqualValues(t, "parse \"§$%&/()\": invalid URL escape \"%&/\"", err.Error())
}

func Test_getCalCmsData_httpRequest_Returns_Data(t *testing.T) {
	teardown := setupTestCal()
	defer teardown()
	respData, _ := os.ReadFile("../samples/calCMS-response.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Add("Content-Type", "application/json")
		w.Write(respData)
	}))
	defer srv.Close()
	cfgCal.CalCms.CmsUrl = srv.URL
	cfgCal.Misc.TestCrawl = true
	cfgCal.Misc.TestDate = "2024/09/24"
	data, err := calCmsService.getCalCmsData()
	assert.Nil(t, err)
	assert.NotNil(t, data)
	assert.EqualValues(t, respData, data)
}

func Test_getCalCmsData_httpRequest_Returns_Error(t *testing.T) {
	teardown := setupTestCal()
	defer teardown()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Add("Content-Type", "application/json")
	}))
	defer srv.Close()
	cfgCal.CalCms.CmsUrl = srv.URL
	cfgCal.Misc.TestCrawl = true
	cfgCal.Misc.TestDate = "2024/09/24"
	data, err := calCmsService.getCalCmsData()
	assert.NotNil(t, err)
	assert.Nil(t, data)
	assert.EqualValues(t, "400 Bad Request", err.Error())
}

func Test_Query_Returns_NoError(t *testing.T) {
	teardown := setupTestCal()
	defer teardown()
	respData, _ := os.ReadFile("../samples/calCMS-response.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Add("Content-Type", "application/json")
		w.Write(respData)
	}))
	defer srv.Close()
	cfgCal.CalCms.CmsUrl = srv.URL
	cfgCal.Misc.TestCrawl = true
	cfgCal.Misc.TestDate = "2024/09/24"
	cfgCal.CalCms.QueryCalCms = true
	err := calCmsService.Query()
	assert.Nil(t, err)
}

func Test_Query_Returns_Error(t *testing.T) {
	teardown := setupTestCal()
	defer teardown()
	respData, _ := os.ReadFile("../samples/calCMS-response_error.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Add("Content-Type", "application/json")
		w.Write(respData)
	}))
	defer srv.Close()
	cfgCal.CalCms.CmsUrl = srv.URL
	cfgCal.Misc.TestCrawl = true
	cfgCal.Misc.TestDate = "2024/09/24"
	cfgCal.CalCms.QueryCalCms = true
	err := calCmsService.Query()
	assert.NotNil(t, err)
	assert.EqualValues(t, "invalid character ':' after object key:value pair", err.Error())
}

func Test_Query_NoQuery_Returns_NoError(t *testing.T) {
	teardown := setupTestCal()
	defer teardown()
	cfgCal.CalCms.QueryCalCms = false
	err := calCmsService.Query()
	assert.Nil(t, err)
}
