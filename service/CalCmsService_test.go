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
	"github.com/johannes-kuhfuss/mairlist-feeder/dto"
	"github.com/johannes-kuhfuss/mairlist-feeder/helper"
	"github.com/johannes-kuhfuss/mairlist-feeder/repositories"
	"github.com/stretchr/testify/assert"
)

var (
	cfgCal        config.AppConfig
	fileRepoCal   repositories.DefaultFileRepository
	calCmsService DefaultCalCmsService
	fi1           domain.FileInfo = domain.FileInfo{
		Path:          "",
		ModTime:       time.Time{},
		Duration:      0,
		StartTime:     time.Time{},
		EndTime:       time.Time{},
		FromCalCMS:    false,
		InfoExtracted: false,
		ScanTime:      time.Time{},
		FolderDate:    folderDateDash,
		RuleMatched:   "",
		EventId:       0,
		CalCmsTitle:   "",
	}
)

const (
	startDate          = "2024-04-01T11:00:00"
	endDate            = "2024-04-01T12:00:00"
	parseDate          = "2006-01-02T15:04:05"
	folderDateDash     = "2024-09-17"
	folderDateSlash    = "2024/09/17"
	calCmsResponseFile = "../samples/calCMS-response.json"
	title              = "my title"
)

func setupTestCal() func() {
	config.InitConfig(config.EnvFile, &cfgCal)
	fileRepoCal = repositories.NewFileRepository(&cfgCal)
	calCmsService = NewCalCmsService(&cfgCal, &fileRepoCal)
	return func() {
		fileRepoCal.DeleteAllData()
	}
}

func TestConvertEventTimeError1ReturnsError(t *testing.T) {
	teardown := setupTestCal()
	defer teardown()
	ev := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartDatetime: "AA:BB",
		EndDatetime:   "CC:DD",
		EventID:       1,
	}
	_, err := calCmsService.convertEventToEntry(ev)
	assert.NotNil(t, err)
	assert.EqualValues(t, "parsing time \"AA:BB\" as \"2006-01-02T15:04:05\": cannot parse \"AA:BB\" as \"2006\"", err.Error())
}

func TestConvertEventTimeError2ReturnsError(t *testing.T) {
	teardown := setupTestCal()
	defer teardown()
	ev := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartDatetime: "11:00",
		EndDatetime:   "CC:DD",
		EventID:       1,
	}
	_, err := calCmsService.convertEventToEntry(ev)
	assert.NotNil(t, err)
	assert.EqualValues(t, "parsing time \"11:00\" as \"2006-01-02T15:04:05\": cannot parse \"11:00\" as \"2006\"", err.Error())
}

func TestConvertEventNoErrorReturnsEntry(t *testing.T) {
	teardown := setupTestCal()
	defer teardown()
	ev := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartDatetime: startDate,
		EndDatetime:   endDate,
		EventID:       12345,
	}
	res, err := calCmsService.convertEventToEntry(ev)
	assert.Nil(t, err)
	t1, _ := time.ParseInLocation(parseDate, startDate, time.Local)
	t2, _ := time.ParseInLocation(parseDate, endDate, time.Local)
	assert.EqualValues(t, t1, res.StartTime)
	assert.EqualValues(t, t2, res.EndTime)
	assert.EqualValues(t, "Test", res.Title)
	assert.EqualValues(t, 12345, res.EventId)
}

func TestGetCalCmsDataForIdEmptyReturnsEmptyList(t *testing.T) {
	teardown := setupTestCal()
	defer teardown()
	res, err := calCmsService.GetCalCmsEventDataForId(1)
	assert.Nil(t, err)
	assert.EqualValues(t, 0, len(res))
}

func TestGetCalCmsDataForIdWrongDataReturnsError(t *testing.T) {
	var events []domain.CalCmsEvent
	teardown := setupTestCal()
	defer teardown()

	event := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartDatetime: startDate,
		EndDatetime:   "CC:DD",
		EventID:       1,
	}
	events = append(events, event)
	data := domain.CalCmsPgmData{
		Events: events,
	}
	calCmsService.insertData(data)

	res, err := calCmsService.GetCalCmsEventDataForId(1)
	assert.NotNil(t, err)
	assert.EqualValues(t, 0, len(res))
	assert.EqualValues(t, "parsing time \"CC:DD\" as \"2006-01-02T15:04:05\": cannot parse \"CC:DD\" as \"2006\"", err.Error())
}

func TestGetCalCmsDataForIdWrongIdReturnsEmpty(t *testing.T) {
	var events []domain.CalCmsEvent
	teardown := setupTestCal()
	defer teardown()

	event := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartDatetime: startDate,
		EndDatetime:   endDate,
		EventID:       2,
	}
	events = append(events, event)
	data := domain.CalCmsPgmData{
		Events: events,
	}
	calCmsService.insertData(data)

	res, err := calCmsService.GetCalCmsEventDataForId(1)
	assert.Nil(t, err)
	assert.EqualValues(t, 0, len(res))
}

func TestGetCalCmsDataForIdOneElementReturnsData(t *testing.T) {
	var events []domain.CalCmsEvent
	teardown := setupTestCal()
	defer teardown()

	event := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartDatetime: startDate,
		EndDatetime:   endDate,
		EventID:       1,
	}
	events = append(events, event)
	data := domain.CalCmsPgmData{
		Events: events,
	}
	calCmsService.insertData(data)

	res, err := calCmsService.GetCalCmsEventDataForId(1)
	assert.Nil(t, err)
	assert.EqualValues(t, 1, len(res))
	assert.EqualValues(t, time.Duration(1*time.Hour), res[0].Duration)
}

func TestGetCalCmsDataForIdTwoElementsReturnsOne(t *testing.T) {
	var events []domain.CalCmsEvent
	teardown := setupTestCal()
	defer teardown()

	event1 := domain.CalCmsEvent{
		FullTitle:     "Test1",
		StartDatetime: startDate,
		EndDatetime:   endDate,
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

	res, err := calCmsService.GetCalCmsEventDataForId(1)
	assert.Nil(t, err)
	assert.EqualValues(t, 1, len(res))
	assert.EqualValues(t, time.Duration(1*time.Hour), res[0].Duration)
}

func TestGetCalCmsDataForHourEmptyReturnsEmptyList(t *testing.T) {
	teardown := setupTestCal()
	defer teardown()
	res, err := calCmsService.GetCalCmsEntriesForHour("12:00")
	assert.Nil(t, err)
	assert.EqualValues(t, 0, len(res))
}

func TestGetCalCmsDataForHourOneElementReturnsData(t *testing.T) {
	var events []domain.CalCmsEvent
	teardown := setupTestCal()
	defer teardown()

	event := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartDatetime: startDate,
		StartTimeName: "11:00",
		EndDatetime:   endDate,
		EventID:       1,
	}
	events = append(events, event)
	data := domain.CalCmsPgmData{
		Events: events,
	}
	calCmsService.insertData(data)

	res, err := calCmsService.GetCalCmsEntriesForHour("11")
	assert.Nil(t, err)
	assert.EqualValues(t, 1, len(res))
	assert.EqualValues(t, time.Duration(1*time.Hour), res[0].Duration)
}

func TestCheckCalCmsDataWrongDataReturnsError(t *testing.T) {
	var events []domain.CalCmsEvent
	teardown := setupTestCal()
	defer teardown()

	event := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartDatetime: startDate,
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

	_, err := calCmsService.checkCalCmsEventData(fi)

	assert.NotNil(t, err)
	assert.EqualValues(t, "parsing time \"CC:DD\" as \"2006-01-02T15:04:05\": cannot parse \"CC:DD\" as \"2006\"", err.Error())
}

func setupEvents() {
	var events []domain.CalCmsEvent
	event := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartDatetime: startDate,
		EndDatetime:   endDate,
		EventID:       1,
	}
	events = append(events, event)
	data := domain.CalCmsPgmData{
		Events: events,
	}
	calCmsService.insertData(data)
}

func TestCheckCalCmsDataDifferingDatesReturnsError(t *testing.T) {
	teardown := setupTestCal()
	defer teardown()
	setupEvents()

	_, err := calCmsService.checkCalCmsEventData(fi1)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "file has different date (2024-09-17) than calCms data")
}

func TestCheckCalCmsDataNoMatchingOnSameDayDataReturnsError(t *testing.T) {
	teardown := setupTestCal()
	defer teardown()
	setupEvents()

	calCmsService.Cfg.Misc.TestCrawl = true
	calCmsService.Cfg.Misc.TestDate = folderDateSlash

	_, err := calCmsService.checkCalCmsEventData(fi1)

	assert.NotNil(t, err)
	assert.EqualValues(t, "no Id 0 in calCMS", err.Error())
}

func TestCheckCalCmsDataDoubleMatchReturnsError(t *testing.T) {
	var events []domain.CalCmsEvent
	teardown := setupTestCal()
	defer teardown()

	event1 := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartDatetime: startDate,
		EndDatetime:   endDate,
		EventID:       1,
	}
	event2 := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartDatetime: endDate,
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
		FolderDate:    folderDateDash,
		RuleMatched:   "",
		EventId:       1,
		CalCmsTitle:   "",
	}

	calCmsService.Cfg.Misc.TestCrawl = true
	calCmsService.Cfg.Misc.TestDate = folderDateSlash

	_, err := calCmsService.checkCalCmsEventData(fi)

	assert.NotNil(t, err)
	assert.EqualValues(t, "multiple matches in calCMS", err.Error())
}

func TestCheckCalCmsDataIsLiveReturnsData(t *testing.T) {
	var events []domain.CalCmsEvent
	teardown := setupTestCal()
	defer teardown()

	event := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartDatetime: startDate,
		EndDatetime:   endDate,
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
		FolderDate:    folderDateDash,
		RuleMatched:   "",
		EventId:       1,
		CalCmsTitle:   "",
	}
	calCmsService.Cfg.Misc.TestCrawl = true
	calCmsService.Cfg.Misc.TestDate = folderDateSlash

	res, err := calCmsService.checkCalCmsEventData(fi)

	assert.Nil(t, err)
	assert.EqualValues(t, event.FullTitle, res.Title)
	st1, _ := time.ParseInLocation(parseDate, event.StartDatetime, time.Local)
	st2, _ := time.ParseInLocation(parseDate, event.EndDatetime, time.Local)
	assert.EqualValues(t, fmt.Sprintf("%02d:%02d", st1.Hour(), st1.Minute()), res.StartTime.Format("15:04"))
	assert.EqualValues(t, fmt.Sprintf("%02d:%02d", st2.Hour(), st2.Minute()), res.EndTime.Format("15:04"))
	assert.EqualValues(t, res.Live, 1)
}

func TestCheckCalCmsDataDataOkReturnsData(t *testing.T) {
	var events []domain.CalCmsEvent
	teardown := setupTestCal()
	defer teardown()

	event := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartDatetime: startDate,
		EndDatetime:   endDate,
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
		FolderDate:    folderDateDash,
		RuleMatched:   "",
		EventId:       1,
		CalCmsTitle:   "",
	}

	calCmsService.Cfg.Misc.TestCrawl = true
	calCmsService.Cfg.Misc.TestDate = folderDateSlash

	res, err := calCmsService.checkCalCmsEventData(fi)

	assert.Nil(t, err)
	assert.EqualValues(t, event.FullTitle, res.Title)
	st1, _ := time.ParseInLocation(parseDate, event.StartDatetime, time.Local)
	st2, _ := time.ParseInLocation(parseDate, event.EndDatetime, time.Local)
	assert.EqualValues(t, fmt.Sprintf("%02d:%02d", st1.Hour(), st1.Minute()), res.StartTime.Format("15:04"))
	assert.EqualValues(t, fmt.Sprintf("%02d:%02d", st2.Hour(), st2.Minute()), res.EndTime.Format("15:04"))
}

func TestEnrichFileInformationNoFilesReturnsZero(t *testing.T) {
	teardown := setupTestCal()
	defer teardown()
	n := calCmsService.EnrichFileInformation()
	assert.EqualValues(t, 0, n.TotalCount)
}

func TestEnrichFileInformationOneFileReturnsEnriched(t *testing.T) {
	var events []domain.CalCmsEvent
	teardown := setupTestCal()
	defer teardown()

	event := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartDatetime: startDate,
		EndDatetime:   endDate,
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
		FolderDate:    folderDateDash,
		RuleMatched:   "",
		EventId:       1234,
		CalCmsTitle:   "",
		FileType:      "Audio",
	}
	fileRepoCal.Store(fi)

	calCmsService.Cfg.Misc.TestCrawl = true
	calCmsService.Cfg.Misc.TestDate = folderDateSlash

	n := calCmsService.EnrichFileInformation()

	assert.EqualValues(t, 1, n.TotalCount)
	assert.EqualValues(t, 1, n.AudioCount)
	assert.EqualValues(t, 0, n.StreamCount)
}

func TestEnrichFileInformationOneStreamReturnsEnriched(t *testing.T) {
	var events []domain.CalCmsEvent
	teardown := setupTestCal()
	defer teardown()

	event := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartDatetime: startDate,
		EndDatetime:   endDate,
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
		FolderDate:    folderDateDash,
		RuleMatched:   "",
		EventId:       1234,
		CalCmsTitle:   "",
		FileType:      "Stream",
		StreamId:      55,
	}
	fileRepoCal.Store(fi)

	calCmsService.Cfg.Misc.TestCrawl = true
	calCmsService.Cfg.Misc.TestDate = folderDateSlash

	n := calCmsService.EnrichFileInformation()

	assert.EqualValues(t, 1, n.TotalCount)
	assert.EqualValues(t, 1, n.StreamCount)
	assert.EqualValues(t, 0, n.AudioCount)
}

func TestGetCalCmsDataWrongUrlReturnsError(t *testing.T) {
	teardown := setupTestCal()
	defer teardown()
	cfgCal.CalCms.CmsUrl = "§$%&/()"
	data, err := calCmsService.getCalCmsEventData()
	assert.Nil(t, data)
	assert.NotNil(t, err)
	assert.EqualValues(t, "parse \"§$%&/()\": invalid URL escape \"%&/\"", err.Error())
}

func TestGetCalCmsDatahttpRequestReturnsData(t *testing.T) {
	teardown := setupTestCal()
	defer teardown()
	respData, _ := os.ReadFile(calCmsResponseFile)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Add("Content-Type", "application/json")
		w.Write(respData)
	}))
	defer srv.Close()
	cfgCal.CalCms.CmsUrl = srv.URL
	cfgCal.Misc.TestCrawl = true
	cfgCal.Misc.TestDate = folderDateSlash
	data, err := calCmsService.getCalCmsEventData()
	assert.Nil(t, err)
	assert.NotNil(t, data)
	assert.EqualValues(t, respData, data)
}

func TestGetCalCmsDatahttpRequestReturnsError(t *testing.T) {
	teardown := setupTestCal()
	defer teardown()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Add("Content-Type", "application/json")
	}))
	defer srv.Close()
	cfgCal.CalCms.CmsUrl = srv.URL
	cfgCal.Misc.TestCrawl = true
	cfgCal.Misc.TestDate = folderDateSlash
	data, err := calCmsService.getCalCmsEventData()
	assert.NotNil(t, err)
	assert.Nil(t, data)
	assert.EqualValues(t, "400 Bad Request", err.Error())
}

func TestQueryReturnsNoError(t *testing.T) {
	teardown := setupTestCal()
	defer teardown()
	respData, _ := os.ReadFile(calCmsResponseFile)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Add("Content-Type", "application/json")
		w.Write(respData)
	}))
	defer srv.Close()
	cfgCal.CalCms.CmsUrl = srv.URL
	cfgCal.Misc.TestCrawl = true
	cfgCal.Misc.TestDate = folderDateSlash
	cfgCal.CalCms.QueryCalCms = true
	err := calCmsService.Query()
	assert.Nil(t, err)
}

func TestQueryReturnsError(t *testing.T) {
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
	cfgCal.Misc.TestDate = folderDateSlash
	cfgCal.CalCms.QueryCalCms = true
	err := calCmsService.Query()
	assert.NotNil(t, err)
	assert.EqualValues(t, "invalid character ':' after object key:value pair", err.Error())
}

func TestQueryNoQueryReturnsNoError(t *testing.T) {
	teardown := setupTestCal()
	defer teardown()
	cfgCal.CalCms.QueryCalCms = false
	err := calCmsService.Query()
	assert.Nil(t, err)
}

func TestCalcCalCmsEndDateWrongDateReturnsErros(t *testing.T) {
	startDate := "sdf"
	endDate, err := calcCalCmsEndDate(startDate)
	assert.NotNil(t, err)
	assert.EqualValues(t, "", endDate)
	assert.EqualValues(t, "parsing time \"sdf\" as \"2006-01-02\": cannot parse \"sdf\" as \"2006\"", err.Error())
}

func TestCalcCalCmsEndDateDateReturnsNextDay(t *testing.T) {
	endDate, err := calcCalCmsEndDate(folderDateDash)
	assert.Nil(t, err)
	assert.EqualValues(t, "2024-09-18", endDate)
}

func TestParseDurationReturnsDuration(t *testing.T) {
	dur := parseDuration("05:00:02")
	assert.EqualValues(t, "300.0", dur)
}

func TestParseDurationShortReturnsNA(t *testing.T) {
	dur := parseDuration("abc")
	assert.EqualValues(t, "N/A", dur)
}

func TestParseDurationNoDurationReturnsNA(t *testing.T) {
	dur := parseDuration("ab:cd:ef")
	assert.EqualValues(t, "N/A", dur)
}

func TestGetEventsReturnsdata(t *testing.T) {
	teardown := setupTestCal()
	defer teardown()
	respData, _ := os.ReadFile(calCmsResponseFile)
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
	ev, err := calCmsService.GetEvents()
	assert.Nil(t, err)
	assert.EqualValues(t, 8, len(ev))
	assert.EqualValues(t, "Morgenmagazin - der Freien Radios", ev[0].Title)
}

func TestCheckHashNoFilesReturnsFalse(t *testing.T) {
	files := domain.FileList{}
	i, h := checkHash(&files)
	assert.EqualValues(t, false, i)
	assert.EqualValues(t, false, h)
}

func TestCheckHashOneFileReturnsFalse(t *testing.T) {
	files := domain.FileList{}
	fi1 := domain.FileInfo{}
	files = append(files, fi1)
	i, h := checkHash(&files)
	assert.EqualValues(t, false, i)
	assert.EqualValues(t, false, h)
}

func TestCheckHashTwoFilesNoChecksumReturnsFalse(t *testing.T) {
	files := domain.FileList{}
	fi1 := domain.FileInfo{}
	fi2 := domain.FileInfo{}
	files = append(files, fi1)
	files = append(files, fi2)
	i, h := checkHash(&files)
	assert.EqualValues(t, false, i)
	assert.EqualValues(t, false, h)
}

func TestCheckHashTwoFilesDifferentChecksumReturnsFalse(t *testing.T) {
	files := domain.FileList{}
	fi1 := domain.FileInfo{
		Checksum: "A",
	}
	fi2 := domain.FileInfo{
		Checksum: "B",
	}
	files = append(files, fi1)
	files = append(files, fi2)
	i, h := checkHash(&files)
	assert.EqualValues(t, false, i)
	assert.EqualValues(t, true, h)
}

func TestCheckHashTwoFilesSameChecksumReturnsTrue(t *testing.T) {
	files := domain.FileList{}
	fi1 := domain.FileInfo{
		Checksum: "A",
	}
	fi2 := domain.FileInfo{
		Checksum: "A",
	}
	files = append(files, fi1)
	files = append(files, fi2)
	i, h := checkHash(&files)
	assert.EqualValues(t, true, i)
	assert.EqualValues(t, true, h)
}

func TestExtractFileInfoNoFilesReturnsNA(t *testing.T) {
	files := domain.FileList{}
	s, d, f := extractFileInfo(&files, false)
	assert.EqualValues(t, "N/A", s)
	assert.EqualValues(t, "N/A", d)
	assert.EqualValues(t, "N/A", f)
}

func TestExtractFileInfoOneFilesNoHashReturnsDuration(t *testing.T) {
	files := domain.FileList{}
	fi1 := domain.FileInfo{
		Checksum: "A",
		Duration: 60.0,
	}
	files = append(files, fi1)
	s, d, f := extractFileInfo(&files, false)
	assert.EqualValues(t, "Present", s)
	assert.EqualValues(t, "1.0", d)
	assert.EqualValues(t, "Manual", f)
}

func TestExtractFileInfoOnecalCmsFilesNoHashReturnsDuration(t *testing.T) {
	files := domain.FileList{}
	fi1 := domain.FileInfo{
		Checksum:   "A",
		Duration:   60.0,
		FromCalCMS: true,
		EventId:    5,
	}
	files = append(files, fi1)
	s, d, f := extractFileInfo(&files, false)
	assert.EqualValues(t, "Present", s)
	assert.EqualValues(t, "1.0", d)
	assert.EqualValues(t, "calCMS", f)
}

func TestExtractFileInfoTwoFilesNoHashReturnsNoDuration(t *testing.T) {
	files := domain.FileList{}
	fi1 := domain.FileInfo{
		Checksum: "A",
		Duration: 60.0,
	}
	fi2 := domain.FileInfo{
		Checksum: "B",
		Duration: 60.0,
	}
	files = append(files, fi1)
	files = append(files, fi2)
	s, d, f := extractFileInfo(&files, false)
	assert.EqualValues(t, "Multiple", s)
	assert.EqualValues(t, "N/A", d)
	assert.EqualValues(t, "N/A", f)
}

func TestExtractFileInfoTwoFilesHashMissingReturnsNoDuration(t *testing.T) {
	files := domain.FileList{}
	fi1 := domain.FileInfo{
		Duration: 60.0,
	}
	fi2 := domain.FileInfo{
		Duration: 60.0,
	}
	files = append(files, fi1)
	files = append(files, fi2)
	s, d, f := extractFileInfo(&files, true)
	assert.EqualValues(t, "Multiple", s)
	assert.EqualValues(t, "N/A", d)
	assert.EqualValues(t, "N/A", f)
}

func TestExtractFileInfoTwoDifferentFilesWithHashReturnsDifferent(t *testing.T) {
	files := domain.FileList{}
	fi1 := domain.FileInfo{
		Checksum: "A",
		Duration: 60.0,
	}
	fi2 := domain.FileInfo{
		Checksum: "B",
		Duration: 60.0,
	}
	files = append(files, fi1)
	files = append(files, fi2)
	s, d, f := extractFileInfo(&files, true)
	assert.EqualValues(t, "Multiple (different)", s)
	assert.EqualValues(t, "N/A", d)
	assert.EqualValues(t, "N/A", f)
}

func TestExtractFileInfoTwoIdenticalFilesWithHashReturnsSame(t *testing.T) {
	files := domain.FileList{}
	fi1 := domain.FileInfo{
		Checksum: "A",
		Duration: 60.0,
	}
	fi2 := domain.FileInfo{
		Checksum: "A",
		Duration: 60.0,
	}
	files = append(files, fi1)
	files = append(files, fi2)
	s, d, f := extractFileInfo(&files, true)
	assert.EqualValues(t, "Multiple (identical)", s)
	assert.EqualValues(t, "1.0", d)
	assert.EqualValues(t, "N/A", f)
}

func TestSetCalCmsQueryStateSuccess(t *testing.T) {
	teardown := setupTestCal()
	defer teardown()
	calCmsService.setCalCmsQueryState(true)
	assert.Contains(t, calCmsService.Cfg.RunTime.LastCalCmsState, "Succeeded")
}

func TestSetCalCmsQueryStateFailure(t *testing.T) {
	teardown := setupTestCal()
	defer teardown()
	calCmsService.setCalCmsQueryState(false)
	assert.Contains(t, calCmsService.Cfg.RunTime.LastCalCmsState, "Failed")
}

func TestMergeInfoNotFromCalCms(t *testing.T) {
	oi := domain.FileInfo{
		FromCalCMS: false,
	}
	ci := dto.CalCmsEntry{}
	ni, ct := mergeInfo(oi, ci)
	assert.EqualValues(t, 1, ct.TotalCount)
	assert.EqualValues(t, true, ni.FromCalCMS)
	assert.EqualValues(t, true, ni.CalCmsInfoExtracted)
	assert.False(t, ni.EventIsLive)
}

func TestMergeInfoStartTimesDiffer(t *testing.T) {
	oi := domain.FileInfo{
		FromCalCMS: true,
		StartTime:  time.Date(2024, 11, 11, 1, 2, 3, 0, time.Local),
	}
	ci := dto.CalCmsEntry{
		StartTime: time.Date(2024, 11, 11, 1, 2, 4, 0, time.Local),
	}
	ni, ct := mergeInfo(oi, ci)
	assert.EqualValues(t, 1, ct.TotalCount)
	assert.EqualValues(t, time.Date(2024, 11, 11, 1, 2, 4, 0, time.Local), ni.StartTime)
	assert.False(t, ni.EventIsLive)
}

func TestMergeInfoAudio(t *testing.T) {
	oi := domain.FileInfo{
		FromCalCMS: true,
		FileType:   "Audio",
	}
	oi.StartTime = time.Date(2024, 11, 11, 1, 2, 3, 0, time.Local)
	ci := dto.CalCmsEntry{
		Title:     title,
		StartTime: time.Date(2024, 11, 11, 1, 2, 3, 0, time.Local),
		EndTime:   time.Date(2024, 11, 11, 2, 2, 3, 0, time.Local),
		Duration:  0,
		EventId:   0,
		Live:      0,
	}
	ni, ct := mergeInfo(oi, ci)
	assert.EqualValues(t, 1, ct.TotalCount)
	assert.EqualValues(t, 1, ct.AudioCount)
	assert.EqualValues(t, 0, ct.StreamCount)
	assert.EqualValues(t, time.Date(2024, 11, 11, 1, 2, 3, 0, time.Local), ni.StartTime)
	assert.EqualValues(t, time.Date(2024, 11, 11, 2, 2, 3, 0, time.Local), ni.EndTime)
	assert.EqualValues(t, title, ni.CalCmsTitle)
	assert.False(t, ni.EventIsLive)
}

func TestMergeInfoStream(t *testing.T) {
	oi := domain.FileInfo{
		FromCalCMS: true,
		FileType:   "Stream",
		StreamId:   9,
	}
	oi.StartTime = time.Date(2024, 11, 11, 1, 2, 3, 0, time.Local)
	ci := dto.CalCmsEntry{
		Title:     title,
		StartTime: time.Date(2024, 11, 11, 1, 2, 3, 0, time.Local),
		EndTime:   time.Date(2024, 11, 11, 2, 2, 3, 0, time.Local),
		Duration:  time.Duration(60 * time.Second),
		EventId:   3,
		Live:      0,
	}
	ni, ct := mergeInfo(oi, ci)
	assert.EqualValues(t, 1, ct.TotalCount)
	assert.EqualValues(t, 0, ct.AudioCount)
	assert.EqualValues(t, 1, ct.StreamCount)
	assert.EqualValues(t, 60.0, ni.Duration)
	assert.False(t, ni.EventIsLive)
}

func TestMergeInfoIsLive(t *testing.T) {
	oi := domain.FileInfo{
		FromCalCMS: true,
		StartTime:  time.Date(2024, 11, 11, 1, 2, 3, 0, time.Local),
	}
	ci := dto.CalCmsEntry{
		StartTime: time.Date(2024, 11, 11, 1, 2, 3, 0, time.Local),
		Live:      1,
	}
	ni, ct := mergeInfo(oi, ci)
	assert.EqualValues(t, 1, ct.TotalCount)
	assert.True(t, ni.EventIsLive)
}

func TestMergeLiveWasReset(t *testing.T) {
	oi := domain.FileInfo{
		FromCalCMS:  true,
		StartTime:   time.Date(2024, 11, 11, 1, 2, 3, 0, time.Local),
		EventIsLive: true,
	}
	ci := dto.CalCmsEntry{
		StartTime: time.Date(2024, 11, 11, 1, 2, 3, 0, time.Local),
		Live:      0,
	}
	ni, ct := mergeInfo(oi, ci)
	assert.EqualValues(t, 1, ct.TotalCount)
	assert.False(t, ni.EventIsLive)
}
