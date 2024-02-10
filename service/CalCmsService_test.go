package service

import (
	"testing"
	"time"

	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/domain"
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
	t1, _ := time.Parse("15:04", "11:00")
	t2, _ := time.Parse("15:04", "12:00")
	assert.EqualValues(t, t1, res.StartTime)
	assert.EqualValues(t, t2, res.EndTime)
	assert.EqualValues(t, "Test", res.Title)
	assert.EqualValues(t, 12345, res.EventId)
}


