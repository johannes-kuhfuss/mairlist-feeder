package dto

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
	repo repositories.DefaultFileRepository
	cfg  config.AppConfig
)

func setupTest(t *testing.T) func() {
	repo = repositories.NewFileRepository(&cfg)
	return func() {
	}
}

func Test_GetFiles_NoFiles_ReturnsEmpty(t *testing.T) {
	teardown := setupTest(t)
	defer teardown()
	res := GetFiles(&repo)
	assert.EqualValues(t, 0, len(res))
}

func Test_GetFiles_TwoFiles_ReturnsFileData(t *testing.T) {
	teardown := setupTest(t)
	defer teardown()
	fi1 := domain.FileInfo{
		Path:                "A",
		ModTime:             time.Time{},
		Duration:            3600,
		StartTime:           helper.TimeFromHourAndMinute(11, 0),
		EndTime:             time.Time{},
		FromCalCMS:          false,
		InfoExtracted:       false,
		ScanTime:            time.Time{},
		FolderDate:          "2023-12-31",
		RuleMatched:         "None",
		EventId:             1,
		CalCmsTitle:         "",
		CalCmsInfoExtracted: false,
	}
	fi2 := domain.FileInfo{
		Path:                "B",
		ModTime:             time.Time{},
		Duration:            3600,
		StartTime:           time.Time{},
		EndTime:             helper.TimeFromHourAndMinute(13, 0),
		FromCalCMS:          false,
		InfoExtracted:       false,
		ScanTime:            time.Time{},
		FolderDate:          "2023-12-31",
		RuleMatched:         "None",
		EventId:             1,
		CalCmsTitle:         "",
		CalCmsInfoExtracted: false,
	}
	repo.Store(fi1)
	repo.Store(fi2)

	res := GetFiles(&repo)

	assert.EqualValues(t, 2, len(res))
	assert.EqualValues(t, "2023-12-31", res[0].FolderDate)
	assert.EqualValues(t, "2023-12-31", res[1].FolderDate)
}
