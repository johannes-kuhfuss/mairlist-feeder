package service

import (
	"testing"
	"time"

	metrics "github.com/johannes-kuhfuss/mairlist-feeder/Metrics"
	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/domain"
	"github.com/johannes-kuhfuss/mairlist-feeder/repositories"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

var (
	cfgClean  config.AppConfig
	cleanSvc  DefaultCleanService
	cleanRepo repositories.DefaultFileRepository
)

func setupTestClean() func() {
	registry := prometheus.NewRegistry()
	config.InitConfig(config.EnvFile, &cfgClean)
	metrics.InitMetrics(&cfgClean, registry)
	cleanRepo = repositories.NewFileRepository(&cfgClean)
	cleanSvc = NewCleanService(&cfgClean, &cleanRepo)
	return func() {
		cleanRepo.DeleteAllData()
		metrics.UnregisterMetrics(&cfgClean, registry)
	}
}

func TestIsYesterdayOrOlderIsOlderReturnsTrue(t *testing.T) {
	testDate := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
	b, e := isYesterdayOrOlder(testDate)
	assert.Nil(t, e)
	assert.EqualValues(t, true, b)
}

func TestIsYesterdayOrOlderIsWayOlderReturnsTrue(t *testing.T) {
	testDate := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC).AddDate(0, -1, 0)
	b, e := isYesterdayOrOlder(testDate)
	assert.Nil(t, e)
	assert.EqualValues(t, true, b)
}

func TestIsYesterdayOrOlderIsNotOlderReturnsFalse(t *testing.T) {
	testDate := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC)
	b, e := isYesterdayOrOlder(testDate)
	assert.Nil(t, e)
	assert.EqualValues(t, false, b)
}

func TestIsYesterdayOrOlderWrongDateReturnsFalse(t *testing.T) {
	b, e := isYesterdayOrOlder(time.Time{})
	assert.NotNil(t, e)
	assert.EqualValues(t, false, b)
}

func TestRunCleanNoFilesReturnsZero(t *testing.T) {
	teardown := setupTestClean()
	defer teardown()
	n, e := cleanSvc.CleanRun()
	assert.Nil(t, e)
	assert.EqualValues(t, 0, n)
}

func TestRunCleanOneCurrentFileReturnsZero(t *testing.T) {
	teardown := setupTestClean()
	defer teardown()
	fi1 := domain.FileInfo{
		Path:       "A",
		FolderDate: domain.NormalizeDate(time.Now()),
	}
	cleanRepo.Store(fi1)
	n, e := cleanSvc.CleanRun()
	f := cleanRepo.Size()
	assert.Nil(t, e)
	assert.EqualValues(t, 0, n)
	assert.EqualValues(t, 1, f)
}

func TestRunCleanOneOldFileReturnsOne(t *testing.T) {
	teardown := setupTestClean()
	defer teardown()
	fi1 := domain.FileInfo{
		Path:       "A",
		FolderDate: domain.NormalizeDate(time.Now().AddDate(0, 0, -1)),
	}
	cleanRepo.Store(fi1)
	f1 := cleanRepo.Size()
	n, e := cleanSvc.CleanRun()
	f2 := cleanRepo.Size()
	assert.Nil(t, e)
	assert.EqualValues(t, 1, n)
	assert.EqualValues(t, 1, f1)
	assert.EqualValues(t, 0, f2)
}

func TestRunCleanFileWrongDateReturnsError(t *testing.T) {
	teardown := setupTestClean()
	defer teardown()
	fi1 := domain.FileInfo{
		Path:       "A",
		FolderDate: time.Time{},
	}
	cleanRepo.Store(fi1)
	n, e := cleanSvc.CleanRun()
	f := cleanRepo.Size()
	assert.NotNil(t, e)
	assert.EqualValues(t, 0, n)
	assert.EqualValues(t, 1, f)
}

func TestCleanNoFilesReturnsZero(t *testing.T) {
	teardown := setupTestClean()
	defer teardown()
	cleanSvc.Clean()
	assert.EqualValues(t, 0, cleanSvc.Cfg.RunTime.FilesCleaned)
}

func TestCleanOneFileReturnsOne(t *testing.T) {
	teardown := setupTestClean()
	defer teardown()
	fi1 := domain.FileInfo{
		Path:       "A",
		FolderDate: domain.NormalizeDate(time.Now().AddDate(0, 0, -2)),
	}
	cleanRepo.Store(fi1)
	cleanSvc.Clean()
	assert.EqualValues(t, 1, cleanSvc.Cfg.RunTime.FilesCleaned)
}
