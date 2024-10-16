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

func setupTest() func() {
	repo = NewFileRepository(&cfg)
	return func() {
	}
}

func TestEmptyListIsEmpty(t *testing.T) {
	teardown := setupTest()
	defer teardown()

	assert.EqualValues(t, 0, repo.Size())
}

func TestGetOnEmptyList(t *testing.T) {
	teardown := setupTest()
	defer teardown()

	res := repo.Get("B")

	assert.Nil(t, res)
}

func TestGetAllOnEmptyList(t *testing.T) {
	teardown := setupTest()
	defer teardown()

	res := repo.GetAll()

	assert.Nil(t, res)
}

func TestAddItemWithEmptyPath(t *testing.T) {
	teardown := setupTest()
	defer teardown()

	fi := domain.FileInfo{}
	err := repo.Store(fi)

	assert.NotNil(t, err)
	assert.EqualValues(t, "cannot add item with empty path to list", err.Error())
}

func TestAddAndGet(t *testing.T) {
	teardown := setupTest()
	defer teardown()

	fi := domain.FileInfo{
		Path:     "A",
		Duration: 1.0,
	}
	err := repo.Store(fi)

	res := repo.Get("A")

	assert.Nil(t, err)
	assert.NotNil(t, res)
	assert.EqualValues(t, "A", res.Path)
	assert.EqualValues(t, 1.0, res.Duration)
}

func TestAddAndGetAll(t *testing.T) {
	teardown := setupTest()
	defer teardown()

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

func TestDeleteEmpty(t *testing.T) {
	teardown := setupTest()
	defer teardown()

	err := repo.Delete("A")

	assert.NotNil(t, err)
	assert.EqualValues(t, "item does not exist", err.Error())
}

func TestDeleteItem(t *testing.T) {
	teardown := setupTest()
	defer teardown()

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

func TestGetForHourEmpty(t *testing.T) {
	teardown := setupTest()
	defer teardown()

	res := repo.GetForHour("13")

	assert.Nil(t, res)
}

func TestGetForHourNoResult(t *testing.T) {
	teardown := setupTest()
	defer teardown()

	fi := domain.FileInfo{
		Path:       "A",
		StartTime:  helper.TimeFromHourAndMinute(12, 0),
		FolderDate: strings.Replace(helper.GetTodayFolder(false, ""), "/", "-", -1),
	}
	repo.Store(fi)

	res := repo.GetForHour("13")

	assert.Nil(t, res)
}

func TestGetForHourOneResult(t *testing.T) {
	teardown := setupTest()
	defer teardown()

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

func TestGetForHourTwoResults(t *testing.T) {
	teardown := setupTest()
	defer teardown()

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

func TestSaveToDiskAndLoad(t *testing.T) {
	teardown := setupTest()
	defer teardown()

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

	repo.SaveToDisk("test.dta")
	repo.DeleteAllData()
	sizeBefore := repo.Size()
	repo.LoadFromDisk("test.dta")
	sizeAfter := repo.Size()

	assert.EqualValues(t, 0, sizeBefore)
	assert.EqualValues(t, 3, sizeAfter)
	os.Remove("test.dta")
}

func TestDeleteAll(t *testing.T) {
	teardown := setupTest()
	defer teardown()

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

func TestNewFilesFalse(t *testing.T) {
	teardown := setupTest()
	defer teardown()

	fi1 := domain.FileInfo{
		Path:          "A",
		StartTime:     helper.TimeFromHourAndMinute(11, 0),
		InfoExtracted: true,
	}
	repo.Store(fi1)
	newFiles := repo.NewFiles()

	assert.EqualValues(t, false, newFiles)
}

func TestNewFilesTrue(t *testing.T) {
	teardown := setupTest()
	defer teardown()

	fi1 := domain.FileInfo{
		Path:          "A",
		StartTime:     helper.TimeFromHourAndMinute(11, 0),
		InfoExtracted: false,
	}
	repo.Store(fi1)
	newFiles := repo.NewFiles()

	assert.EqualValues(t, true, newFiles)
}

func Test_AudioSize_Returns_NumberOfAudioFiles(t *testing.T) {
	teardown := setupTest()
	defer teardown()

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

func Test_StreamSize_Returns_NumberOfStreamFiles(t *testing.T) {
	teardown := setupTest()
	defer teardown()

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

func Test_EventIdExists_NoEvents_Returns_False(t *testing.T) {
	teardown := setupTest()
	defer teardown()
	n := repo.EventIdExists(1)
	assert.EqualValues(t, 0, n)
}

func Test_EventIdExists_NoMatchingEvents_Returns_False(t *testing.T) {
	teardown := setupTest()
	defer teardown()
	fi1 := domain.FileInfo{
		Path:    "A",
		EventId: 2,
	}
	repo.Store(fi1)
	n := repo.EventIdExists(1)
	assert.EqualValues(t, 0, n)
}

func Test_EventIdExists_MatchingEvent_Returns_True(t *testing.T) {
	teardown := setupTest()
	defer teardown()
	fi1 := domain.FileInfo{
		Path:    "A",
		EventId: 2,
	}
	repo.Store(fi1)
	n := repo.EventIdExists(2)
	assert.EqualValues(t, 1, n)
}

func Test_EventIdExists_MultipleEvents_Returns_True(t *testing.T) {
	teardown := setupTest()
	defer teardown()
	fi1 := domain.FileInfo{
		Path:    "A",
		EventId: 2,
	}
	fi2 := domain.FileInfo{
		Path:    "B",
		EventId: 2,
	}
	repo.Store(fi1)
	repo.Store(fi2)
	n := repo.EventIdExists(2)
	assert.EqualValues(t, 2, n)
}
