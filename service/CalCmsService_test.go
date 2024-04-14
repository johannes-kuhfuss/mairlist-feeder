package service

import (
	"fmt"
	"testing"
	"time"

	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/domain"
	"github.com/johannes-kuhfuss/mairlist-feeder/helper"
	"github.com/johannes-kuhfuss/mairlist-feeder/repositories"
	"github.com/stretchr/testify/assert"
)

var (
	cfgA          config.AppConfig
	fileRepoA     repositories.DefaultFileRepository
	calCmsService DefaultCalCmsService
)

func setupTestA() func() {
	config.InitConfig(config.EnvFile, &cfgA)
	fileRepoA = repositories.NewFileRepository(&cfgA)
	calCmsService = NewCalCmsService(&cfgA, &fileRepo)
	return func() {
	}
}

func Test_convertToEntry_TimeError1_ReturnsError(t *testing.T) {
	teardown := setupTestA()
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
	teardown := setupTestA()
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
	teardown := setupTestA()
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
	teardown := setupTestA()
	defer teardown()
	res, err := calCmsService.GetCalCmsDataForId(1)
	assert.Nil(t, err)
	assert.EqualValues(t, 0, len(res))
}

func Test_GetCalCmsDataForId_WrongData_ReturnsError(t *testing.T) {
	var events []domain.CalCmsEvent
	teardown := setupTestA()
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
	teardown := setupTestA()
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
	teardown := setupTestA()
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
	teardown := setupTestA()
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
	teardown := setupTestA()
	defer teardown()
	res, err := calCmsService.GetCalCmsDataForHour("12:00")
	assert.Nil(t, err)
	assert.EqualValues(t, 0, len(res))
}

func Test_GetCalCmsDataForHour_OneElement_ReturnsData(t *testing.T) {
	var events []domain.CalCmsEvent
	teardown := setupTestA()
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
	teardown := setupTestA()
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

func Test_checkCalCmsData_NoMatchingData_ReturnsError(t *testing.T) {
	var events []domain.CalCmsEvent
	teardown := setupTestA()
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
		FolderDate:    "",
		RuleMatched:   "",
		EventId:       0,
		CalCmsTitle:   "",
	}

	_, err := calCmsService.checkCalCmsData(fi)

	assert.NotNil(t, err)
	assert.EqualValues(t, "no such id in calCMS", err.Error())
}

func Test_checkCalCmsData_DoubleMatch_ReturnsError(t *testing.T) {
	var events []domain.CalCmsEvent
	teardown := setupTestA()
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
		FolderDate:    "",
		RuleMatched:   "",
		EventId:       1,
		CalCmsTitle:   "",
	}

	_, err := calCmsService.checkCalCmsData(fi)

	assert.NotNil(t, err)
	assert.EqualValues(t, "multiple matches in calCMS", err.Error())
}

func Test_checkCalCmsData_IsLive_ReturnsError(t *testing.T) {
	var events []domain.CalCmsEvent
	teardown := setupTestA()
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
		FolderDate:    "",
		RuleMatched:   "",
		EventId:       1,
		CalCmsTitle:   "",
	}

	_, err := calCmsService.checkCalCmsData(fi)

	assert.NotNil(t, err)
	assert.EqualValues(t, "event is live in calCMS", err.Error())
}

func Test_checkCalCmsData_DataOk_ReturnsData(t *testing.T) {
	var events []domain.CalCmsEvent
	teardown := setupTestA()
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
		FolderDate:    "",
		RuleMatched:   "",
		EventId:       1,
		CalCmsTitle:   "",
	}

	res, err := calCmsService.checkCalCmsData(fi)

	assert.Nil(t, err)
	assert.EqualValues(t, event.FullTitle, res.Title)
	st1, _ := time.ParseInLocation("2006-01-02T15:04:05", event.StartDatetime, time.Local)
	st2, _ := time.ParseInLocation("2006-01-02T15:04:05", event.EndDatetime, time.Local)
	assert.EqualValues(t, fmt.Sprintf("%02d:%02d", st1.Hour(), st1.Minute()), res.StartTime.Format("15:04"))
	assert.EqualValues(t, fmt.Sprintf("%02d:%02d", st2.Hour(), st2.Minute()), res.EndTime.Format("15:04"))
}
