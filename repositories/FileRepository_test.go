package repositories

import (
	"os"
	"strings"
	"testing"

	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/domain"
	"github.com/johannes-kuhfuss/mairlist-feeder/helper"
	"github.com/stretchr/testify/assert"
)

var (
	cfg  config.AppConfig
	repo DefaultFileRepository
)

func setupTest() {
	repo = NewFileRepository(&cfg)
}

func TestNewFileRepositoryCreatesEmptyList(t *testing.T) {
	setupTest()
	assert.EqualValues(t, 0, repo.Size())
}

func TestGetByPathEmptyListReturnsNil(t *testing.T) {
	setupTest()
	res := repo.GetByPath("B")
	assert.Nil(t, res)
}

func TestGetAllEmptyListReturnsNil(t *testing.T) {
    setupTest()
	res := repo.GetAll()
	assert.Nil(t, res)
}

func TestStoreItemWithEmptyPathReturnsError(t *testing.T) {
	setupTest()
	fi := domain.FileInfo{}
	err := repo.Store(fi)
	assert.NotNil(t, err)
	assert.EqualValues(t, "cannot add item with empty path to list", err.Error())
}

func TestGetByPathCorrectPathReturnsElement(t *testing.T) {
	setupTest()
	fi := domain.FileInfo{
		Path:     "A",
		Duration: 1.0,
	}
	err := repo.Store(fi)
	res := repo.GetByPath("A")
	assert.Nil(t, err)
	assert.NotNil(t, res)
	assert.EqualValues(t, "A", res.Path)
	assert.EqualValues(t, 1.0, res.Duration)
}

func TestGetAllReturnsAllElements(t *testing.T) {
	setupTest()
	fi1 := domain.FileInfo{
		Path:     "A",
		Duration: 2.0,
	}
	fi2 := domain.FileInfo{
		Path:     "B",
		Duration: 2.0,
	}
	repo.Store(fi1)
	repo.Store(fi2)
	size := repo.Size()
	res := repo.GetAll()
	el1 := (*res)[0]
	el2 := (*res)[1]
	assert.NotNil(t, size)
	assert.EqualValues(t, 2, size)
	assert.EqualValues(t, 2, len(*res))
	assert.EqualValues(t, 2.0, el1.Duration)
	assert.EqualValues(t, 2.0, el2.Duration)
}

func TestDeleteNonExistingElementReturnsError(t *testing.T) {
	setupTest()
	err := repo.Delete("A")
	assert.NotNil(t, err)
	assert.EqualValues(t, "item with path A does not exist", err.Error())
}

func TestDeleteExistingElementDeletesElement(t *testing.T) {
	setupTest()
	fi := domain.FileInfo{
		Path:     "A",
		Duration: 1.0,
	}
	repo.Store(fi)
	sizeBefore := repo.Size()
	err := repo.Delete("A")
	sizeAfter := repo.Size()
	assert.Nil(t, err)
	assert.EqualValues(t, 1, sizeBefore)
	assert.EqualValues(t, 0, sizeAfter)
}

func TestGetForHourEmptyListReturnsNil(t *testing.T) {
	setupTest()
	res := repo.GetForHour("13")
	assert.Nil(t, res)
}

func TestGetForHourNoMatchReturnsNil(t *testing.T) {
	setupTest()
	fi := domain.FileInfo{
		Path:       "A",
		StartTime:  helper.TimeFromHourAndMinute(12, 0),
		FolderDate: strings.Replace(helper.GetTodayFolder(false, ""), "/", "-", -1),
	}
	repo.Store(fi)
	res := repo.GetForHour("13")
	assert.Nil(t, res)
}

func TestGetForHourOneMatchReturnsElement(t *testing.T) {
	setupTest()
	fi := domain.FileInfo{
		Path:       "A",
		StartTime:  helper.TimeFromHourAndMinute(12, 0),
		FolderDate: strings.Replace(helper.GetTodayFolder(false, ""), "/", "-", -1),
	}
	repo.Store(fi)
	res := repo.GetForHour("12")
	el := (*res)[0]
	assert.NotNil(t, *res)
	assert.EqualValues(t, 1, len(*res))
	assert.EqualValues(t, "A", el.Path)
	assert.EqualValues(t, helper.TimeFromHourAndMinute(12, 0), el.StartTime)
}

func TestGetForHourTwoMatchesReturnsElements(t *testing.T) {
	setupTest()
	fi1 := domain.FileInfo{
		Path:       "A",
		StartTime:  helper.TimeFromHourAndMinute(11, 0),
		FolderDate: strings.Replace(helper.GetTodayFolder(false, ""), "/", "-", -1),
	}
	fi2 := domain.FileInfo{
		Path:       "B",
		StartTime:  helper.TimeFromHourAndMinute(12, 0),
		FolderDate: strings.Replace(helper.GetTodayFolder(false, ""), "/", "-", -1),
	}
	fi3 := domain.FileInfo{
		Path:       "C",
		StartTime:  helper.TimeFromHourAndMinute(12, 30),
		FolderDate: strings.Replace(helper.GetTodayFolder(false, ""), "/", "-", -1),
	}
	repo.Store(fi1)
	repo.Store(fi2)
	repo.Store(fi3)
	res := repo.GetForHour("12")
	assert.NotNil(t, *res)
	assert.EqualValues(t, 2, len(*res))
}

func TestSaveToDiskSavesToDisk(t *testing.T) {
	setupTest()
	fi1 := domain.FileInfo{
		Path:      "A",
		StartTime: helper.TimeFromHourAndMinute(11, 0),
	}
	fi2 := domain.FileInfo{
		Path:      "B",
		StartTime: helper.TimeFromHourAndMinute(12, 0),
	}
	fi3 := domain.FileInfo{
		Path:      "C",
		StartTime: helper.TimeFromHourAndMinute(12, 30),
	}
	repo.Store(fi1)
	repo.Store(fi2)
	repo.Store(fi3)
	err1 := repo.SaveToDisk("test.dta")
	repo.DeleteAllData()
	sizeBefore := repo.Size()
	err2 := repo.LoadFromDisk("test.dta")
	sizeAfter := repo.Size()
	assert.Nil(t, err1)
	assert.Nil(t, err2)
	assert.EqualValues(t, 0, sizeBefore)
	assert.EqualValues(t, 3, sizeAfter)
	os.Remove("test.dta")
}

func TestLoadFromDiskNoFileReturnsError(t *testing.T) {
	setupTest()
	err := repo.LoadFromDisk("no.file")
	assert.NotNil(t, err)
	assert.EqualValues(t, "open no.file: The system cannot find the file specified.", err.Error())
}

func TestLoadFromDiskWrongDataReturnsError(t *testing.T) {
	setupTest()
	os.WriteFile("test.dta", []byte("someBogusData"), 0666)
	err := repo.LoadFromDisk("test.dta")
	assert.NotNil(t, err)
	assert.EqualValues(t, "invalid character 's' looking for beginning of value", err.Error())
	os.Remove("test.dta")
}

func TestDeleteAllDeletesAllElements(t *testing.T) {
	setupTest()
	fi1 := domain.FileInfo{
		Path:      "A",
		StartTime: helper.TimeFromHourAndMinute(11, 0),
	}
	fi2 := domain.FileInfo{
		Path:      "B",
		StartTime: helper.TimeFromHourAndMinute(12, 0),
	}
	repo.Store(fi1)
	repo.Store(fi2)
	sizeBefore := repo.Size()
	repo.DeleteAllData()
	sizeAfter := repo.Size()
	assert.EqualValues(t, 2, sizeBefore)
	assert.EqualValues(t, 0, sizeAfter)
}

func TestNewFilesNoNewFilesReturnsFalse(t *testing.T) {
	setupTest()
	fi1 := domain.FileInfo{
		Path:          "A",
		StartTime:     helper.TimeFromHourAndMinute(11, 0),
		InfoExtracted: true,
	}
	repo.Store(fi1)
	newFiles := repo.NewFiles()
	assert.EqualValues(t, false, newFiles)
}

func TestNewFilesNewFilesReturnsTrue(t *testing.T) {
	setupTest()
	fi1 := domain.FileInfo{
		Path:          "A",
		StartTime:     helper.TimeFromHourAndMinute(11, 0),
		InfoExtracted: false,
	}
	repo.Store(fi1)
	newFiles := repo.NewFiles()
	assert.EqualValues(t, true, newFiles)
}

func TestAudioSizeReturnsNumberOfAudioFiles(t *testing.T) {
	setupTest()
	s1 := repo.AudioSize()
	fi1 := domain.FileInfo{
		Path:      "A",
		StartTime: helper.TimeFromHourAndMinute(11, 0),
		FileType:  "Audio",
	}
	repo.Store(fi1)
	s2 := repo.AudioSize()
	assert.EqualValues(t, 0, s1)
	assert.EqualValues(t, 1, s2)
}

func TestStreamSizeReturnsNumberOfStreamFiles(t *testing.T) {
	setupTest()
	s1 := repo.StreamSize()
	fi1 := domain.FileInfo{
		Path:      "A",
		StartTime: helper.TimeFromHourAndMinute(11, 0),
		FileType:  "Stream",
	}
	repo.Store(fi1)
	s2 := repo.StreamSize()
	assert.EqualValues(t, 0, s1)
	assert.EqualValues(t, 1, s2)
}

func TestGetByEventIdEmptyReturnsNil(t *testing.T) {
	setupTest()
	res := repo.GetByEventId(1)
	assert.Nil(t, res)
}

func TestGetByEventIdNoEventIdReturnsNil(t *testing.T) {
	setupTest()
	fi1 := domain.FileInfo{
		Path:    "A",
		EventId: 2,
	}
	repo.Store(fi1)
	res := repo.GetByEventId(1)
	assert.Nil(t, res)
}

func TestGetByEventIdOneEventIdReturnsOne(t *testing.T) {
	setupTest()
	fi1 := domain.FileInfo{
		Path:    "A",
		EventId: 1,
	}
	repo.Store(fi1)
	res := repo.GetByEventId(1)
	assert.NotNil(t, res)
	assert.EqualValues(t, 1, len(*res))
	assert.EqualValues(t, "A", (*res)[0].Path)
}

func TestGetByEventIdTwoEventIdsReturnsOne(t *testing.T) {
	setupTest()
	fi1 := domain.FileInfo{
		Path:    "A",
		EventId: 1,
	}
	fi2 := domain.FileInfo{
		Path:    "B",
		EventId: 2,
	}
	repo.Store(fi1)
	repo.Store(fi2)
	res := repo.GetByEventId(1)
	assert.NotNil(t, res)
	assert.EqualValues(t, 1, len(*res))
	assert.EqualValues(t, "A", (*res)[0].Path)
}

func TestGetByEventIdTwoEventIdsReturnsTwo(t *testing.T) {
	setupTest()
	fi1 := domain.FileInfo{
		Path:    "A",
		EventId: 1,
	}
	fi2 := domain.FileInfo{
		Path:    "B",
		EventId: 1,
	}
	repo.Store(fi1)
	repo.Store(fi2)
	res := repo.GetByEventId(1)
	assert.NotNil(t, res)
	assert.EqualValues(t, 2, len(*res))
	assert.EqualValues(t, "A", (*res)[0].Path)
	assert.EqualValues(t, "B", (*res)[1].Path)
}
