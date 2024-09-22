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
	cfgCrawl config.AppConfig
	crawlSvc DefaultCrawlService
	repo     repositories.DefaultFileRepository
)

func setupTestCrawl() func() {
	config.InitConfig(config.EnvFile, &cfgCrawl)
	repo = repositories.NewFileRepository(&cfgCrawl)
	crawlSvc = NewCrawlService(&cfgCrawl, &repo, nil)
	return func() {
		repo.DeleteAllData()
	}
}

func Test_parseEventId_WithNumId_ReturnsId(t *testing.T) {
	teardown := setupTestCrawl()
	defer teardown()
	id := crawlSvc.parseEventId("Z:\\sendungen\\2024\\02\\04\\18-00\\2024-02-03_11-31-57-id34067-seniorenradio-CRKaleidoskop2024.mp3")
	assert.EqualValues(t, 34067, id)
}

func Test_parseEventId_NoId_ReturnsZero(t *testing.T) {
	teardown := setupTestCrawl()
	defer teardown()
	id := crawlSvc.parseEventId("Z:\\sendungen\\2024\\02\\04\\18-00\\2024-02-03_11-31-57-seniorenradio-CRKaleidoskop2024.mp3")
	assert.EqualValues(t, 0, id)
}

func Test_parseEventId_NoNumId_ReturnsZero(t *testing.T) {
	teardown := setupTestCrawl()
	defer teardown()
	id := crawlSvc.parseEventId("Z:\\sendungen\\2024\\02\\04\\18-00\\2024-02-03_11-31-57-id34AB067-seniorenradio-CRKaleidoskop2024.mp3")
	assert.EqualValues(t, 0, id)
}

func Test_extractFileInfo_NoFiles_ReturnsZero(t *testing.T) {
	teardown := setupTestCrawl()
	defer teardown()
	n, e := crawlSvc.extractFileInfo()
	assert.Nil(t, e)
	assert.EqualValues(t, 0, n)
}

func Test_extractFileInfo_FileCalCms_ReturnsData(t *testing.T) {
	teardown := setupTestCrawl()
	defer teardown()
	fi1 := domain.FileInfo{
		Path:       "Z:\\sendungen\\2024\\09\\22\\21-00\\test.mp3",
		FolderDate: "2024-09-22",
	}
	repo.Store(fi1)
	n, e := crawlSvc.extractFileInfo()
	fires := repo.Get(fi1.Path)
	assert.Nil(t, e)
	assert.EqualValues(t, 1, n)
	assert.EqualValues(t, true, fires.FromCalCMS)
	assert.EqualValues(t, "folder HH-MM (calCMS)", fires.RuleMatched)
	assert.EqualValues(t, time.Date(2024, time.September, 22, 21, 0, 0, 0, time.Local), fires.StartTime)
}

func Test_extractFileInfo_FileNamingConvention_ReturnsData(t *testing.T) {
	teardown := setupTestCrawl()
	defer teardown()
	fi1 := domain.FileInfo{
		Path:       "Z:\\sendungen\\2024\\09\\22\\2000-2100_sendung-xyz.mp3",
		FolderDate: "2024-09-22",
	}
	repo.Store(fi1)
	n, e := crawlSvc.extractFileInfo()
	fires := repo.Get(fi1.Path)
	assert.Nil(t, e)
	assert.EqualValues(t, 1, n)
	assert.EqualValues(t, false, fires.FromCalCMS)
	assert.EqualValues(t, "file HHMM-HHMM", fires.RuleMatched)
	assert.EqualValues(t, time.Date(2024, time.September, 22, 20, 0, 0, 0, time.Local), fires.StartTime)
	assert.EqualValues(t, time.Date(2024, time.September, 22, 21, 0, 0, 0, time.Local), fires.EndTime)
}

func Test_extractFileInfo_Uploadtool_ReturnsData(t *testing.T) {
	teardown := setupTestCrawl()
	defer teardown()
	fi1 := domain.FileInfo{
		Path:       "Z:\\sendungen\\2024\\09\\22\\UL__1800-1900__sendung-xyz.mp3",
		FolderDate: "2024-09-22",
	}
	repo.Store(fi1)
	n, e := crawlSvc.extractFileInfo()
	fires := repo.Get(fi1.Path)
	assert.Nil(t, e)
	assert.EqualValues(t, 1, n)
	assert.EqualValues(t, false, fires.FromCalCMS)
	assert.EqualValues(t, "Upload Tool", fires.RuleMatched)
	assert.EqualValues(t, time.Date(2024, time.September, 22, 18, 0, 0, 0, time.Local), fires.StartTime)
	assert.EqualValues(t, time.Date(2024, time.September, 22, 19, 0, 0, 0, time.Local), fires.EndTime)
}

func Test_extractFileInfo_AnyFile_ReturnsData(t *testing.T) {
	teardown := setupTestCrawl()
	defer teardown()
	fi1 := domain.FileInfo{
		Path:       "Z:\\sendungen\\2024\\09\\22\\2100_sendung-xyz.mp3",
		FolderDate: "2024-09-22",
	}
	repo.Store(fi1)
	n, e := crawlSvc.extractFileInfo()
	fires := repo.Get(fi1.Path)
	assert.Nil(t, e)
	assert.EqualValues(t, 1, n)
	assert.EqualValues(t, false, fires.FromCalCMS)
	assert.EqualValues(t, "None", fires.RuleMatched)
}

func Test_extractFileInfo_RealFile_ReturnsData(t *testing.T) {
	teardown := setupTestCrawl()
	defer teardown()
	cfgCrawl.Crawl.FfprobePath = "../prog/ffprobe.exe"
	fi1 := domain.FileInfo{
		Path:       "../samples/1600-1700_sine1k.mp3",
		FolderDate: "2024-09-22",
	}
	repo.Store(fi1)
	n, e := crawlSvc.extractFileInfo()
	fires := repo.Get(fi1.Path)
	assert.Nil(t, e)
	assert.EqualValues(t, 1, n)
	assert.EqualValues(t, false, fires.FromCalCMS)
	assert.EqualValues(t, "file HHMM-HHMM", fires.RuleMatched)
	assert.EqualValues(t, 5.041633, fires.Duration)
	assert.EqualValues(t, 34, fires.BitRate)
	assert.EqualValues(t, "MP2/3 (MPEG audio layer 2/3)", fires.FormatName)
}

func Test_convertTime_WrongStartTime_Returns_Error(t *testing.T) {
	ti, e := convertTime("A", "B", "C")
	assert.EqualValues(t, time.Time{}, ti)
	assert.NotNil(t, e)
	assert.EqualValues(t, "strconv.Atoi: parsing \"A\": invalid syntax", e.Error())
}

func Test_convertTime_WrongEndTime_Returns_Error(t *testing.T) {
	ti, e := convertTime("11", "B", "C")
	assert.EqualValues(t, time.Time{}, ti)
	assert.NotNil(t, e)
	assert.EqualValues(t, "strconv.Atoi: parsing \"B\": invalid syntax", e.Error())
}

func Test_convertTime_WrongDate_Returns_Error(t *testing.T) {
	ti, e := convertTime("11", "12", "C")
	assert.EqualValues(t, time.Time{}, ti)
	assert.NotNil(t, e)
	assert.EqualValues(t, "parsing time \"C\" as \"2006-01-02\": cannot parse \"C\" as \"2006\"", e.Error())
}

func Test_convertTime_CorrectValues_Returns_Time(t *testing.T) {
	ti, e := convertTime("11", "12", "2024-09-21")
	assert.Nil(t, e)
	assert.EqualValues(t, time.Date(2024, time.September, 21, 11, 12, 0, 0, time.Local), ti)
}
