package repositories

import (
	"testing"

	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/domain"
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

	assert.NotNil(t, size)
	assert.EqualValues(t, 2, size)
	assert.EqualValues(t, 2, len(*res))
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
