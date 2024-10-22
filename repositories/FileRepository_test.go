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

func Test_NewFileRepository_CreatesEmptyList(t *testing.T) {
	teardown := setupTest()
	defer teardown()

	assert.EqualValues(t, 0, repo.Size())
}

func Test_GetByPath_EmptyList_Returns_Nil(t *testing.T) {
	teardown := setupTest()
	defer teardown()

	res := repo.GetByPath("B")

	assert.Nil(t, res)
}

func Test_GetAll_EmptyList_Returns_Nil(t *testing.T) {
	teardown := setupTest()
	defer teardown()

	res := repo.GetAll()

	assert.Nil(t, res)
}

func Test_Store_ItemWithEmptyPath_Returns_Error(t *testing.T) {
	teardown := setupTest()
	defer teardown()

	fi := domain.FileInfo{}
	err := repo.Store(fi)

	assert.NotNil(t, err)
	assert.EqualValues(t, "cannot add item with empty path to list", err.Error())
}

func Test_GetByPath_CorrectPath_Returns_Element(t *testing.T) {
	teardown := setupTest()
	defer teardown()

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

func Test_GetAll_Returns_AllElements(t *testing.T) {
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

func Test_Delete_NonExistingElement_Returns_Error(t *testing.T) {
	teardown := setupTest()
	defer teardown()

	err := repo.Delete("A")

	assert.NotNil(t, err)
	assert.EqualValues(t, "item with path A does not exist", err.Error())
}

func Test_Delete_ExistingElement_Deletes_Element(t *testing.T) {
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

func Test_GetForHour_EmptyList_Returns_Nil(t *testing.T) {
	teardown := setupTest()
	defer teardown()

	res := repo.GetForHour("13")

	assert.Nil(t, res)
}

func Test_GetForHour_NoMatch_Returns_Nil(t *testing.T) {
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

func Test_GetForHour_OneMatch_Returns_Element(t *testing.T) {
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

func Test_GetForHour_TwoMatches_Returns_Elements(t *testing.T) {
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

func Test_SaveToDisk_SavesToDisk(t *testing.T) {
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

func Test_LoadFromDisk_NoFile_Returns_Error(t *testing.T) {
	teardown := setupTest()
	defer teardown()

	err := repo.LoadFromDisk("no.file")

	assert.NotNil(t, err)
	assert.EqualValues(t, "open no.file: The system cannot find the file specified.", err.Error())
}

func Test_LoadFromDisk_WrongData_Returns_Error(t *testing.T) {
	teardown := setupTest()
	defer teardown()

	os.WriteFile("test.dta", []byte("someBogusData"), 0666)

	err := repo.LoadFromDisk("test.dta")

	assert.NotNil(t, err)
	assert.EqualValues(t, "invalid character 's' looking for beginning of value", err.Error())
	os.Remove("test.dta")
}

func Test_DeleteAll_Deletes_AllElements(t *testing.T) {
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

func Test_NewFiles_NoNewFiles_Returns_False(t *testing.T) {
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

func Test_NewFiles_NewFiles_Returns_True(t *testing.T) {
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

func Test_GetByEventId_Empty_Returns_Nil(t *testing.T) {
	teardown := setupTest()
	defer teardown()

	res := repo.GetByEventId(1)

	assert.Nil(t, res)
}

func Test_GetByEventId_NoEventId_Returns_Nil(t *testing.T) {
	teardown := setupTest()
	defer teardown()
	fi1 := domain.FileInfo{
		Path:    "A",
		EventId: 2,
	}
	repo.Store(fi1)

	res := repo.GetByEventId(1)

	assert.Nil(t, res)
}

func Test_GetByEventId_OneEventId_Returns_One(t *testing.T) {
	teardown := setupTest()
	defer teardown()
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

func Test_GetByEventId_TwoEventIds_Returns_One(t *testing.T) {
	teardown := setupTest()
	defer teardown()
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

func Test_GetByEventId_TwoEventIds_Returns_Two(t *testing.T) {
	teardown := setupTest()
	defer teardown()
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
