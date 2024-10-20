package service

import (
	"os"
	"testing"
	"time"

	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/domain"
	"github.com/johannes-kuhfuss/mairlist-feeder/repositories"
	"github.com/stretchr/testify/assert"
)

var (
	cfgCrawl  config.AppConfig
	crawlSvc  DefaultCrawlService
	crawlRepo repositories.DefaultFileRepository
)

func setupTestCrawl() func() {
	config.InitConfig(config.EnvFile, &cfgCrawl)
	crawlRepo = repositories.NewFileRepository(&cfgCrawl)
	crawlSvc = NewCrawlService(&cfgCrawl, &crawlRepo, nil)
	return func() {
		crawlRepo.DeleteAllData()
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
	assert.EqualValues(t, 0, n.TotalCount)
	assert.EqualValues(t, 0, n.AudioCount)
	assert.EqualValues(t, 0, n.StreamCount)
}

func Test_extractFileInfo_FileCalCms_ReturnsData(t *testing.T) {
	teardown := setupTestCrawl()
	defer teardown()
	fi1 := domain.FileInfo{
		Path:       "Z:\\sendungen\\2024\\09\\22\\21-00\\test.mp3",
		FolderDate: "2024-09-22",
	}
	crawlRepo.Store(fi1)
	n, e := crawlSvc.extractFileInfo()
	fires := crawlRepo.GetByPath(fi1.Path)
	assert.Nil(t, e)
	assert.EqualValues(t, 1, n.TotalCount)
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
	crawlRepo.Store(fi1)
	n, e := crawlSvc.extractFileInfo()
	fires := crawlRepo.GetByPath(fi1.Path)
	assert.Nil(t, e)
	assert.EqualValues(t, 1, n.TotalCount)
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
	crawlRepo.Store(fi1)
	n, e := crawlSvc.extractFileInfo()
	fires := crawlRepo.GetByPath(fi1.Path)
	assert.Nil(t, e)
	assert.EqualValues(t, 1, n.TotalCount)
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
	crawlRepo.Store(fi1)
	n, e := crawlSvc.extractFileInfo()
	fires := crawlRepo.GetByPath(fi1.Path)
	assert.Nil(t, e)
	assert.EqualValues(t, 1, n.TotalCount)
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
	crawlRepo.Store(fi1)
	n, e := crawlSvc.extractFileInfo()
	fires := crawlRepo.GetByPath(fi1.Path)
	assert.Nil(t, e)
	assert.EqualValues(t, 1, n.TotalCount)
	assert.EqualValues(t, 1, n.AudioCount)
	assert.EqualValues(t, 0, n.StreamCount)
	assert.EqualValues(t, false, fires.FromCalCMS)
	assert.EqualValues(t, "file HHMM-HHMM", fires.RuleMatched)
	assert.EqualValues(t, 5.041633, fires.Duration)
	assert.EqualValues(t, 34, fires.BitRate)
	assert.EqualValues(t, "MP2/3 (MPEG audio layer 2/3)", fires.FormatName)
}

func Test_extractFileInfo_Stream_ReturnsData(t *testing.T) {
	teardown := setupTestCrawl()
	defer teardown()

	file := "./temp.stream"
	fi1 := domain.FileInfo{
		Path:       file,
		FolderDate: "2024-09-22",
	}
	crawlRepo.Store(fi1)
	crawlSvc.Cfg.Crawl.StreamMap["test"] = 222
	os.WriteFile(file, []byte("test"), 0644)
	n, e := crawlSvc.extractFileInfo()
	fires := crawlRepo.GetByPath(fi1.Path)
	assert.Nil(t, e)
	assert.EqualValues(t, 1, n.TotalCount)
	assert.EqualValues(t, 0, n.AudioCount)
	assert.EqualValues(t, 1, n.StreamCount)
	assert.EqualValues(t, "Stream", fires.FileType)
	assert.EqualValues(t, "test", fires.StreamName)
	assert.EqualValues(t, 222, fires.StreamId)
	os.Remove(file)
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

func Test_parseTechMd_NoDate_Returns_Error(t *testing.T) {
	_, e := parseTechMd([]byte{})
	assert.NotNil(t, e)
	assert.EqualValues(t, "unexpected end of JSON input", e.Error())
}

func Test_parseTechMd_NoDuration_Returns_Error(t *testing.T) {
	data, _ := os.ReadFile("../samples/ffprobe_nodur.json")
	_, e := parseTechMd(data)
	assert.NotNil(t, e)
	assert.EqualValues(t, "strconv.ParseFloat: parsing \"A\": invalid syntax", e.Error())
}

func Test_parseTechMd_CorrectData_Returns_TechMD(t *testing.T) {
	data, _ := os.ReadFile("../samples/ffprobe_allok.json")
	tm, e := parseTechMd(data)
	assert.Nil(t, e)
	assert.NotNil(t, tm)
	assert.EqualValues(t, 5.034, tm.DurationSec)
	assert.EqualValues(t, 34, tm.BitRate)
	assert.EqualValues(t, "MP2/3 (MPEG audio layer 2/3)", tm.FormatName)
}

func Test_analyzeTechMd_WrongFfprobePath_Returns_Error(t *testing.T) {
	d, e := analyzeTechMd("/here/file", 5, "/here/no/ffprobe")
	assert.Nil(t, d)
	assert.NotNil(t, e)
	assert.EqualValues(t, "exec: \"/here/no/ffprobe\": executable file not found in %PATH%", e.Error())
}

func Test_analyzeTechMd_SampleFile_Returns_TechMd(t *testing.T) {
	d, e := analyzeTechMd("../samples/1600-1700_sine1k.mp3", 5, "../prog/ffprobe.exe")
	assert.Nil(t, e)
	assert.NotNil(t, d)
	assert.EqualValues(t, 5.041633, d.DurationSec)
}

func Test_crawlFolder_NoFiles_Returns_Zero(t *testing.T) {
	teardown := setupTestCrawl()
	defer teardown()

	cfgCrawl.Misc.TestCrawl = true
	cfgCrawl.Misc.TestDate = "2024/09/22"

	n, e := crawlSvc.crawlFolder("../samples/", []string{".mp3"})

	assert.Nil(t, e)
	assert.EqualValues(t, 0, n)
}

func Test_crawlFolder_OneFiles_Returns_One(t *testing.T) {
	teardown := setupTestCrawl()
	defer teardown()

	cfgCrawl.Misc.TestCrawl = true
	cfgCrawl.Misc.TestDate = "2024/09/23"

	n, e := crawlSvc.crawlFolder("../samples/", []string{".mp3"})

	assert.Nil(t, e)
	assert.EqualValues(t, 1, n)
	assert.EqualValues(t, 1, crawlRepo.Size())
}

func Test_analyzeStreamData_NoFile_Returns_Error(t *testing.T) {
	streamMap := make(map[string]int)
	_, _, err := analyzeStreamData("", streamMap)
	assert.NotNil(t, err)
	assert.EqualValues(t, "open : The system cannot find the file specified.", err.Error())
}

func Test_analyzeStreamData_StreamNotFound_Returns_Error(t *testing.T) {
	file := "./temp.txt"
	streamMap := make(map[string]int)
	os.WriteFile(file, []byte("streamX"), 0644)
	_, _, err := analyzeStreamData(file, streamMap)
	assert.NotNil(t, err)
	assert.EqualValues(t, "no such stream configured", err.Error())
	os.Remove(file)
}

func Test_analyzeStreamData_StreamFound_Returns_NameAndId(t *testing.T) {
	file := "./temp.txt"
	streamMap := make(map[string]int)
	streamMap["streamy"] = 55
	os.WriteFile(file, []byte("streamY"), 0644)
	name, id, err := analyzeStreamData(file, streamMap)
	assert.Nil(t, err)
	assert.EqualValues(t, "streamy", name)
	assert.EqualValues(t, 55, id)
	os.Remove(file)
}

func Test_checkForOrphanFiles_NoFiles_Returns_Zero(t *testing.T) {
	teardown := setupTestCrawl()
	defer teardown()
	fr := crawlSvc.checkForOrphanFiles()
	assert.EqualValues(t, 0, fr)
}

func Test_checkForOrphanFiles_OneOrphanFile_Returns_One(t *testing.T) {
	teardown := setupTestCrawl()
	defer teardown()
	fi1 := domain.FileInfo{
		Path:       "../file.txt",
		FolderDate: "2024-09-22",
	}
	crawlRepo.Store(fi1)
	s1 := crawlRepo.Size()
	fr := crawlSvc.checkForOrphanFiles()
	s2 := crawlRepo.Size()
	assert.EqualValues(t, 1, s1)
	assert.EqualValues(t, 1, fr)
	assert.EqualValues(t, 0, s2)
}

func Test_generateHash_NoFile_Returns_Error(t *testing.T) {
	hash, err := generateHash("../no-file")
	assert.EqualValues(t, "", hash)
	assert.NotNil(t, err)
	assert.EqualValues(t, "open ../no-file: The system cannot find the file specified.", err.Error())
}

func Test_generateHash_SampleFile_Returns_Hash(t *testing.T) {
	hash, err := generateHash("../samples/1600-1700_sine1k.mp3")
	assert.Nil(t, err)
	assert.EqualValues(t, "50c2fcde004eea6790580b01c7032f1d", hash)
}
