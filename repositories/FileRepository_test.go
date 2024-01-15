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

func setupTest(t *testing.T) func() {
	repo = NewFileRepository(&cfg)
	return func() {
	}
}

func TestEmptyListIsEmpty(t *testing.T) {
	teardown := setupTest(t)
	defer teardown()

	assert.EqualValues(t, 0, repo.Size())
}

func TestGetOnEmptyList(t *testing.T) {
	teardown := setupTest(t)
	defer teardown()

	res := repo.Get("B")

	assert.Nil(t, res)
}

func TestGetAllOnEmptyList(t *testing.T) {
	teardown := setupTest(t)
	defer teardown()

	res := repo.GetAll()

	assert.Nil(t, res)
}

func TestAddItemWithEmptyPath(t *testing.T) {
	teardown := setupTest(t)
	defer teardown()

	fi := domain.FileInfo{}
	err := repo.Store(fi)

	assert.NotNil(t, err)
	assert.EqualValues(t, "cannot add item with empty path to list", err.Error())
}

func TestAddAndGet(t *testing.T) {
	teardown := setupTest(t)
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
	teardown := setupTest(t)
	defer teardown()

	fi1 := domain.FileInfo{
		Path:     "A",
		Duration: 1.0,
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
	assert.EqualValues(t, "A", el1.Path)
	assert.EqualValues(t, "B", el2.Path)
}

func TestDeleteEmpty(t *testing.T) {
	teardown := setupTest(t)
	defer teardown()

	err := repo.Delete("A")

	assert.NotNil(t, err)
	assert.EqualValues(t, "item does not exist", err.Error())
}

func TestDeleteItem(t *testing.T) {
	teardown := setupTest(t)
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

func TestGetForHourNoResult(t *testing.T) {
	teardown := setupTest(t)
	defer teardown()

	fi := domain.FileInfo{
		Path:       "A",
		StartTime:  "12:00",
		FolderDate: strings.Replace(helper.GetTodayFolder(false, ""), "/", "-", -1),
	}
	repo.Store(fi)

	res := repo.GetForHour("13")

	assert.Nil(t, res)
}

func TestGetForHourOneResult(t *testing.T) {
	teardown := setupTest(t)
	defer teardown()

	fi := domain.FileInfo{
		Path:       "A",
		StartTime:  "12:00",
		FolderDate: strings.Replace(helper.GetTodayFolder(false, ""), "/", "-", -1),
	}
	repo.Store(fi)

	res := repo.GetForHour("12")
	el := (*res)[0]

	assert.NotNil(t, *res)
	assert.EqualValues(t, 1, len(*res))
	assert.EqualValues(t, "A", el.Path)
	assert.EqualValues(t, "12:00", el.StartTime)
}

func TestGetForHourTwoResults(t *testing.T) {
	teardown := setupTest(t)
	defer teardown()

	fi1 := domain.FileInfo{
		Path:       "A",
		StartTime:  "11:00",
		FolderDate: strings.Replace(helper.GetTodayFolder(false, ""), "/", "-", -1),
	}
	fi2 := domain.FileInfo{
		Path:       "B",
		StartTime:  "12:00",
		FolderDate: strings.Replace(helper.GetTodayFolder(false, ""), "/", "-", -1),
	}
	fi3 := domain.FileInfo{
		Path:       "C",
		StartTime:  "12:30",
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
	teardown := setupTest(t)
	defer teardown()

	fi1 := domain.FileInfo{
		Path:      "A",
		StartTime: "11:00",
	}
	fi2 := domain.FileInfo{
		Path:      "B",
		StartTime: "12:00",
	}
	fi3 := domain.FileInfo{
		Path:      "C",
		StartTime: "12:30",
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
