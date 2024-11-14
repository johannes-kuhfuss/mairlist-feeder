// package dto defines the data structures used to exchange information
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
	fis  = domain.FileInfo{
		Path:                "",
		ModTime:             time.Time{},
		Duration:            0,
		StartTime:           time.Time{},
		EndTime:             time.Time{},
		FromCalCMS:          false,
		InfoExtracted:       false,
		ScanTime:            time.Time{},
		FolderDate:          folderDate,
		RuleMatched:         "",
		EventId:             0,
		CalCmsTitle:         "",
		CalCmsInfoExtracted: false,
		BitRate:             123,
		FormatName:          "MyFormat",
		SlotLength:          0,
		FileType:            "",
		StreamId:            0,
		StreamName:          "",
		Checksum:            "",
	}
)

const (
	folderDate = "2023-12-31"
)

func setupTest() {
	repo = repositories.NewFileRepository(&cfg)
}

func TestGetFilesNoFilesReturnsEmpty(t *testing.T) {
	setupTest()
	res := GetFiles(&repo, "")
	assert.EqualValues(t, 0, len(res))
}

func TestGetFilesTwoFilesReturnsFileData(t *testing.T) {
	setupTest()
	fi1 := domain.FileInfo{
		Path:                "A",
		ModTime:             time.Time{},
		Duration:            3600,
		StartTime:           helper.TimeFromHourAndMinute(11, 0),
		EndTime:             time.Time{},
		FromCalCMS:          false,
		InfoExtracted:       false,
		ScanTime:            time.Time{},
		FolderDate:          folderDate,
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
		FolderDate:          folderDate,
		RuleMatched:         "None",
		EventId:             0,
		CalCmsTitle:         "",
		CalCmsInfoExtracted: false,
	}
	repo.Store(fi1)
	repo.Store(fi2)

	res := GetFiles(&repo, "")

	assert.EqualValues(t, 2, len(res))
	assert.EqualValues(t, folderDate, res[0].FolderDate)
	assert.EqualValues(t, folderDate, res[1].FolderDate)
}

func TestBuildCalCmsInfoReturnsInfo1(t *testing.T) {
	fi1 := domain.FileInfo{
		Path:                "A",
		ModTime:             time.Time{},
		Duration:            3600,
		StartTime:           helper.TimeFromHourAndMinute(11, 0),
		EndTime:             time.Time{},
		FromCalCMS:          true,
		InfoExtracted:       true,
		ScanTime:            time.Time{},
		FolderDate:          folderDate,
		RuleMatched:         "None",
		EventId:             1,
		CalCmsTitle:         "myTitle",
		CalCmsInfoExtracted: true,
	}
	s := buildCalCmsInfo(fi1)
	assert.EqualValues(t, "Yes, Yes, \"myTitle\"", s)
}

func TestBuildCalCmsInfoReturnsInfo2(t *testing.T) {
	fi1 := domain.FileInfo{
		Path:                "A",
		ModTime:             time.Time{},
		Duration:            3600,
		StartTime:           helper.TimeFromHourAndMinute(11, 0),
		EndTime:             time.Time{},
		FromCalCMS:          false,
		InfoExtracted:       true,
		ScanTime:            time.Time{},
		FolderDate:          folderDate,
		RuleMatched:         "None",
		EventId:             1,
		CalCmsTitle:         "",
		CalCmsInfoExtracted: false,
	}
	s := buildCalCmsInfo(fi1)
	assert.EqualValues(t, "No, No, None", s)
}

func TestBuildTechMdDefault(t *testing.T) {
	info := buildTechMd(fis)
	assert.EqualValues(t, "N/A", info)
}

func TestBuildTechMdAudioChecksum(t *testing.T) {
	fis.FormatName = "MyFormat"
	fis.BitRate = 123
	fis.FileType = "Audio"
	fis.Checksum = "ABC"
	info := buildTechMd(fis)
	assert.EqualValues(t, "MyFormat @ 123kbps [ABC]", info)
}

func TestBuildTechMdStream(t *testing.T) {
	fis.FileType = "Stream"
	fis.StreamId = 123
	fis.StreamName = "MyStream"
	info := buildTechMd(fis)
	assert.EqualValues(t, "Stream MyStream with Id 123", info)
}
