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

	res := repo.GetFileData("B")

	assert.Nil(t, res)
}

func TestAddAndGet(t *testing.T) {
	teardown := setupTest(t)
	defer teardown()

	fi := domain.FileInfo{
		Path:     "A",
		Duration: 1.0,
	}
	repo.Store(fi)

	res := repo.GetFileData("A")

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
