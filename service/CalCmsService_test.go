package service

import (
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

func setupTestA(t *testing.T) func() {
	config.InitConfig(config.EnvFile, &cfgA)
	fileRepoA = repositories.NewFileRepository(&cfgA)
	calCmsService = NewCalCmsService(&cfgA, &fileRepo)
	return func() {
	}
}

func Test_convertToEntry_TimeError1_ReturnsError(t *testing.T) {
	teardown := setupTestA(t)
	defer teardown()
	ev := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartTimeName: "AA:BB",
		EndTimeName:   "CC:DD",
		EventID:       "ABCDE",
	}
	_, err := calCmsService.convertToEntry(ev)
	assert.NotNil(t, err)
	assert.EqualValues(t, "parsing time \"AA:BB\" as \"15:04\": cannot parse \"AA:BB\" as \"15\"", err.Error())
}

func Test_convertToEntry_TimeError2_ReturnsError(t *testing.T) {
	teardown := setupTestA(t)
	defer teardown()
	ev := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartTimeName: "11:00",
		EndTimeName:   "CC:DD",
		EventID:       "ABCDE",
	}
	_, err := calCmsService.convertToEntry(ev)
	assert.NotNil(t, err)
	assert.EqualValues(t, "parsing time \"CC:DD\" as \"15:04\": cannot parse \"CC:DD\" as \"15\"", err.Error())
}

func Test_convertToEntry_IdError_ReturnsError(t *testing.T) {
	teardown := setupTestA(t)
	defer teardown()
	ev := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartTimeName: "11:00",
		EndTimeName:   "12:00",
		EventID:       "ABCDE",
	}
	_, err := calCmsService.convertToEntry(ev)
	assert.NotNil(t, err)
	assert.EqualValues(t, "strconv.Atoi: parsing \"ABCDE\": invalid syntax", err.Error())
}

func Test_convertToEntry_NoError_ReturnsEntry(t *testing.T) {
	teardown := setupTestA(t)
	defer teardown()
	ev := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartTimeName: "11:00",
		EndTimeName:   "12:00",
		EventID:       "12345",
	}
	res, err := calCmsService.convertToEntry(ev)
	assert.Nil(t, err)
	t1, _ := time.ParseInLocation("15:04", "11:00", time.Local)
	t2, _ := time.ParseInLocation("15:04", "12:00", time.Local)
	t1 = t1.AddDate(1, 0, 0)
	t2 = t2.AddDate(1, 0, 0)
	assert.EqualValues(t, t1, res.StartTime)
	assert.EqualValues(t, t2, res.EndTime)
	assert.EqualValues(t, "Test", res.Title)
	assert.EqualValues(t, 12345, res.EventId)
}

func Test_GetCalCmsDataForId_Empty_ReturnsEmptyList(t *testing.T) {
	teardown := setupTestA(t)
	defer teardown()
	res, err := calCmsService.GetCalCmsDataForId(1)
	assert.Nil(t, err)
	assert.EqualValues(t, 0, len(res))
}

func Test_GetCalCmsDataForId_WrongData_ReturnsError(t *testing.T) {
	var events []domain.CalCmsEvent
	teardown := setupTestA(t)
	defer teardown()

	event := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartTimeName: "11:00",
		EndTimeName:   "CC:DD",
		EventID:       "1",
	}
	events = append(events, event)
	data := domain.CalCmsPgmData{
		Events: events,
	}
	calCmsService.insertData(data)

	res, err := calCmsService.GetCalCmsDataForId(1)
	assert.NotNil(t, err)
	assert.EqualValues(t, 0, len(res))
	assert.EqualValues(t, "parsing time \"CC:DD\" as \"15:04\": cannot parse \"CC:DD\" as \"15\"", err.Error())
}

func Test_GetCalCmsDataForId_WrongId_ReturnsEmpty(t *testing.T) {
	var events []domain.CalCmsEvent
	teardown := setupTestA(t)
	defer teardown()

	event := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartTimeName: "11:00",
		EndTimeName:   "12:00",
		EventID:       "2",
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
	teardown := setupTestA(t)
	defer teardown()

	event := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartTimeName: "11:00",
		EndTimeName:   "12:00",
		EventID:       "1",
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
	teardown := setupTestA(t)
	defer teardown()

	event1 := domain.CalCmsEvent{
		FullTitle:     "Test1",
		StartTimeName: "11:00",
		EndTimeName:   "12:00",
		EventID:       "1",
	}
	event2 := domain.CalCmsEvent{
		FullTitle:     "Test2",
		StartTimeName: "13:00",
		EndTimeName:   "15:00",
		EventID:       "2",
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
	teardown := setupTestA(t)
	defer teardown()
	res, err := calCmsService.GetCalCmsDataForHour("12:00")
	assert.Nil(t, err)
	assert.EqualValues(t, 0, len(res))
}

func Test_GetCalCmsDataForHour_OneElement_ReturnsData(t *testing.T) {
	var events []domain.CalCmsEvent
	teardown := setupTestA(t)
	defer teardown()

	event := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartTimeName: "11:00",
		EndTimeName:   "12:00",
		EventID:       "1",
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
	teardown := setupTestA(t)
	defer teardown()

	event := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartTimeName: "11:00",
		EndTimeName:   "CC:DD",
		EventID:       "1",
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
	assert.EqualValues(t, "parsing time \"CC:DD\" as \"15:04\": cannot parse \"CC:DD\" as \"15\"", err.Error())
}

func Test_checkCalCmsData_NoMatchingData_ReturnsError(t *testing.T) {
	var events []domain.CalCmsEvent
	teardown := setupTestA(t)
	defer teardown()

	event := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartTimeName: "11:00",
		EndTimeName:   "12:00",
		EventID:       "1",
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
	teardown := setupTestA(t)
	defer teardown()

	event1 := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartTimeName: "11:00",
		EndTimeName:   "12:00",
		EventID:       "1",
	}
	event2 := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartTimeName: "12:00",
		EndTimeName:   "13:00",
		EventID:       "1",
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
	teardown := setupTestA(t)
	defer teardown()

	event := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartTimeName: "11:00",
		EndTimeName:   "12:00",
		EventID:       "1",
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

func Test_checkCalCmsData_StartTimeDiff_ReturnsError(t *testing.T) {
	var events []domain.CalCmsEvent
	teardown := setupTestA(t)
	defer teardown()

	event := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartTimeName: "11:00",
		EndTimeName:   "12:00",
		EventID:       "1",
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
		StartTime:     helper.TimeFromHourAndMinute(13, 0),
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
	assert.EqualValues(t, "start time difference between file and calCMS", err.Error())
}

func Test_checkCalCmsData_DataOk_ReturnsData(t *testing.T) {
	var events []domain.CalCmsEvent
	teardown := setupTestA(t)
	defer teardown()

	event := domain.CalCmsEvent{
		FullTitle:     "Test",
		StartTimeName: "11:00",
		EndTimeName:   "12:00",
		EventID:       "1",
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
		StartTime:     helper.TimeFromHourAndMinute(11, 0),
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
	assert.EqualValues(t, event.StartTimeName, res.StartTime.Format("15:04"))
	assert.EqualValues(t, event.EndTimeName, res.EndTime.Format("15:04"))
}
