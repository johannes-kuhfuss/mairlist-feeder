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

const (
	folderDate      = "2024-09-22"
	audioSampleFile = "../samples/1600-1700_sine1k.mp3"
	sampleFolder    = "../samples/"
)

var parsedFolderDate = domain.MustParseFolderDate(folderDate)

func setupTestCrawl() func() {
	config.InitConfig(config.EnvFile, &cfgCrawl)
	crawlRepo = repositories.NewFileRepository(&cfgCrawl)
	crawlSvc = NewCrawlService(&cfgCrawl, &crawlRepo, nil)
	return func() {
		crawlRepo.DeleteAllData()
	}
}

func TestParseEventIdWithNumIdReturnsId(t *testing.T) {
	teardown := setupTestCrawl()
	defer teardown()
	id := crawlSvc.parseEventId("Z:\\sendungen\\2024\\02\\04\\18-00\\2024-02-03_11-31-57-id34067-seniorenradio-CRKaleidoskop2024.mp3")
	assert.EqualValues(t, 34067, id)
}

func TestParseEventIdNoIdReturnsZero(t *testing.T) {
	teardown := setupTestCrawl()
	defer teardown()
	id := crawlSvc.parseEventId("Z:\\sendungen\\2024\\02\\04\\18-00\\2024-02-03_11-31-57-seniorenradio-CRKaleidoskop2024.mp3")
	assert.EqualValues(t, 0, id)
}

func TestParseEventIdNoNumIdReturnsZero(t *testing.T) {
	teardown := setupTestCrawl()
	defer teardown()
	id := crawlSvc.parseEventId("Z:\\sendungen\\2024\\02\\04\\18-00\\2024-02-03_11-31-57-id34AB067-seniorenradio-CRKaleidoskop2024.mp3")
	assert.EqualValues(t, 0, id)
}

func TestExtractFileInfoNoFilesReturnsZero(t *testing.T) {
	teardown := setupTestCrawl()
	defer teardown()
	n, e := crawlSvc.extractFileInfo()
	assert.Nil(t, e)
	assert.EqualValues(t, 0, n.TotalCount)
	assert.EqualValues(t, 0, n.AudioCount)
	assert.EqualValues(t, 0, n.StreamCount)
}

func TestExtractFileInfoFileCalCmsReturnsData(t *testing.T) {
	teardown := setupTestCrawl()
	defer teardown()
	fi1 := domain.FileInfo{
		Path:       "Z:\\sendungen\\2024\\09\\22\\21-00\\test.mp3",
		FolderDate: parsedFolderDate,
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

func TestExtractFileInfoFileNamingConventionReturnsData(t *testing.T) {
	teardown := setupTestCrawl()
	defer teardown()
	fi1 := domain.FileInfo{
		Path:       "Z:\\sendungen\\2024\\09\\22\\2000-2100_sendung-xyz.mp3",
		FolderDate: parsedFolderDate,
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

func TestExtractFileInfoAnyFileReturnsData(t *testing.T) {
	teardown := setupTestCrawl()
	defer teardown()
	fi1 := domain.FileInfo{
		Path:       "Z:\\sendungen\\2024\\09\\22\\2100_sendung-xyz.mp3",
		FolderDate: parsedFolderDate,
	}
	crawlRepo.Store(fi1)
	n, e := crawlSvc.extractFileInfo()
	fires := crawlRepo.GetByPath(fi1.Path)
	assert.Nil(t, e)
	assert.EqualValues(t, 1, n.TotalCount)
	assert.EqualValues(t, false, fires.FromCalCMS)
	assert.EqualValues(t, "None", fires.RuleMatched)
}

func TestExtractFileInfoRealFileReturnsData(t *testing.T) {
	teardown := setupTestCrawl()
	defer teardown()
	cfgCrawl.Crawl.FfprobePath = "../prog/ffprobe.exe"
	fi1 := domain.FileInfo{
		Path:       audioSampleFile,
		FolderDate: parsedFolderDate,
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
	assert.EqualValues(t, 5*time.Second, fires.Duration.Round(time.Second))
	assert.EqualValues(t, 34, fires.BitRate)
	assert.EqualValues(t, "MP2/3 (MPEG audio layer 2/3)", fires.FormatName)
}

func TestExtractFileInfoStreamReturnsData(t *testing.T) {
	teardown := setupTestCrawl()
	defer teardown()

	file := "./temp.stream"
	fi1 := domain.FileInfo{
		Path:       file,
		FolderDate: parsedFolderDate,
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
	assert.EqualValues(t, domain.FileTypeStream, fires.FileType)
	assert.EqualValues(t, "test", fires.StreamName)
	assert.EqualValues(t, 222, fires.StreamId)
	os.Remove(file)
}

func TestConvertTimeWrongStartTimeReturnsError(t *testing.T) {
	ti, e := convertTime("A", "B", parsedFolderDate)
	assert.EqualValues(t, time.Time{}, ti)
	assert.NotNil(t, e)
	assert.EqualValues(t, "strconv.Atoi: parsing \"A\": invalid syntax", e.Error())
}

func TestConvertTimeWrongEndTimeReturnsError(t *testing.T) {
	ti, e := convertTime("11", "B", parsedFolderDate)
	assert.EqualValues(t, time.Time{}, ti)
	assert.NotNil(t, e)
	assert.EqualValues(t, "strconv.Atoi: parsing \"B\": invalid syntax", e.Error())
}

func TestConvertTimeWrongDateReturnsError(t *testing.T) {
	ti, e := convertTime("11", "12", time.Time{})
	assert.EqualValues(t, time.Date(1, time.January, 1, 11, 12, 0, 0, time.Local), ti)
	assert.Nil(t, e)
}

func TestConvertTimeCorrectValuesReturnsTime(t *testing.T) {
	ti, e := convertTime("11", "12", domain.MustParseFolderDate("2024-09-21"))
	assert.Nil(t, e)
	assert.EqualValues(t, time.Date(2024, time.September, 21, 11, 12, 0, 0, time.Local), ti)
}

func TestParseTechMdNoDateReturnsError(t *testing.T) {
	_, e := parseTechMd([]byte{})
	assert.NotNil(t, e)
	assert.EqualValues(t, "unexpected end of JSON input", e.Error())
}

func TestParseTechMdNoDurationReturnsError(t *testing.T) {
	data, _ := os.ReadFile("../samples/ffprobe_nodur.json")
	_, e := parseTechMd(data)
	assert.NotNil(t, e)
	assert.EqualValues(t, "strconv.ParseFloat: parsing \"A\": invalid syntax", e.Error())
}

func TestParseTechMdCorrectDataReturnsTechMD(t *testing.T) {
	data, _ := os.ReadFile("../samples/ffprobe_allok.json")
	tm, e := parseTechMd(data)
	assert.Nil(t, e)
	assert.NotNil(t, tm)
	assert.EqualValues(t, 5034*time.Millisecond, tm.Duration.Round(time.Millisecond))
	assert.EqualValues(t, 34, tm.BitRate)
	assert.EqualValues(t, "MP2/3 (MPEG audio layer 2/3)", tm.FormatName)
}

func TestAnalyzeTechMdWrongFfprobePathReturnsError(t *testing.T) {
	d, e := analyzeTechMd("/here/file", 5, "/here/no/ffprobe")
	assert.Nil(t, d)
	assert.NotNil(t, e)
	assert.Contains(t, e.Error(), "executable file not found")
}

func TestAnalyzeTechMdSampleFileReturnsTechMd(t *testing.T) {
	d, e := analyzeTechMd(audioSampleFile, 5, "../prog/ffprobe.exe")
	assert.Nil(t, e)
	assert.NotNil(t, d)
	assert.EqualValues(t, 5*time.Second, d.Duration.Round(time.Second))
}

func TestCrawlFolderNoFilesReturnsZero(t *testing.T) {
	teardown := setupTestCrawl()
	defer teardown()

	cfgCrawl.Misc.TestCrawl = true
	cfgCrawl.Misc.TestDate = "2024/09/22"

	n, e := crawlSvc.crawlFolder(sampleFolder, []string{".mp3"})

	assert.Nil(t, e)
	assert.EqualValues(t, 0, n)
}

func TestCrawlFolderOneFilesReturnsOne(t *testing.T) {
	teardown := setupTestCrawl()
	defer teardown()

	cfgCrawl.Misc.TestCrawl = true
	cfgCrawl.Misc.TestDate = "2024/09/23"

	n, e := crawlSvc.crawlFolder(sampleFolder, []string{".mp3"})

	assert.Nil(t, e)
	assert.EqualValues(t, 1, n)
	assert.EqualValues(t, 1, crawlRepo.Size())
}

func TestCrawlFolderSecondCrawl(t *testing.T) {
	teardown := setupTestCrawl()
	defer teardown()

	cfgCrawl.Misc.TestCrawl = true
	cfgCrawl.Misc.TestDate = "2024/09/23"

	n1, e1 := crawlSvc.crawlFolder(sampleFolder, []string{".mp3"})
	s1 := crawlRepo.Size()
	n2, e2 := crawlSvc.crawlFolder(sampleFolder, []string{".mp3"})
	s2 := crawlRepo.Size()

	assert.Nil(t, e1)
	assert.Nil(t, e2)
	assert.EqualValues(t, 1, n1)
	assert.EqualValues(t, 0, n2)
	assert.EqualValues(t, 1, s1)
	assert.EqualValues(t, 1, s2)
}

func TestAnalyzeStreamDataNoFileReturnsError(t *testing.T) {
	streamMap := make(map[string]int)
	_, _, err := analyzeStreamData("", streamMap)
	assert.NotNil(t, err)
	assert.True(t, os.IsNotExist(err))
}

func TestAnalyzeStreamDataStreamNotFoundReturnsError(t *testing.T) {
	file := "./temp.txt"
	streamMap := make(map[string]int)
	os.WriteFile(file, []byte("streamX"), 0644)
	_, _, err := analyzeStreamData(file, streamMap)
	assert.NotNil(t, err)
	assert.EqualValues(t, "no such stream configured", err.Error())
	os.Remove(file)
}

func TestAnalyzeStreamDataStreamFoundReturnsNameAndId(t *testing.T) {
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

func TestAnalyzeStreamDataMixedCaseMapKeyReturnsNameAndId(t *testing.T) {
	file := "./temp.txt"
	streamMap := make(map[string]int)
	streamMap["StreamY"] = 55
	os.WriteFile(file, []byte("streamy"), 0644)
	name, id, err := analyzeStreamData(file, streamMap)
	assert.Nil(t, err)
	assert.EqualValues(t, "StreamY", name)
	assert.EqualValues(t, 55, id)
	os.Remove(file)
}

func TestFolderDateFromPathCorrectPathReturnsFolderDate(t *testing.T) {
	folderDate, err := folderDateFromPath("Z:\\sendungen\\2024\\09\\22\\21-00\\test.mp3", "Z:\\sendungen")
	assert.Nil(t, err)
	assert.EqualValues(t, domain.MustParseFolderDate("2024-09-22"), folderDate)
}

func TestFolderDateFromPathShortPathReturnsError(t *testing.T) {
	folderDate, err := folderDateFromPath("Z:\\sendungen\\2024\\test.mp3", "Z:\\sendungen")
	assert.EqualValues(t, time.Time{}, folderDate)
	assert.NotNil(t, err)
}

func TestCheckForOrphanFilesNoFilesReturnsZero(t *testing.T) {
	teardown := setupTestCrawl()
	defer teardown()
	fr := crawlSvc.checkForOrphanFiles()
	assert.EqualValues(t, 0, fr)
}

func TestCheckForOrphanFilesOneOrphanFileReturnsOne(t *testing.T) {
	teardown := setupTestCrawl()
	defer teardown()
	fi1 := domain.FileInfo{
		Path:       "../file.txt",
		FolderDate: domain.MustParseFolderDate("2024-09-22"),
	}
	crawlRepo.Store(fi1)
	s1 := crawlRepo.Size()
	fr := crawlSvc.checkForOrphanFiles()
	s2 := crawlRepo.Size()
	assert.EqualValues(t, 1, s1)
	assert.EqualValues(t, 1, fr)
	assert.EqualValues(t, 0, s2)
}

func TestGenerateHashNoFileReturnsError(t *testing.T) {
	hash, err := generateHash("../no-file")
	assert.EqualValues(t, "", hash)
	assert.NotNil(t, err)
	assert.True(t, os.IsNotExist(err))
}

func TestGenerateHashSampleFileReturnsHash(t *testing.T) {
	hash, err := generateHash(audioSampleFile)
	assert.Nil(t, err)
	assert.EqualValues(t, "50c2fcde004eea6790580b01c7032f1d", hash)
}

func TestGenHashesNoFilesReturnsZero(t *testing.T) {
	teardown := setupTestCrawl()
	defer teardown()
	hc, err := crawlSvc.GenHashes()
	assert.Nil(t, err)
	assert.EqualValues(t, 0, hc)
}

func TestGenHashesOneFileReturnsOne(t *testing.T) {
	teardown := setupTestCrawl()
	defer teardown()
	fi1 := domain.FileInfo{
		Path: audioSampleFile,
	}
	crawlRepo.Store(fi1)
	hc, err := crawlSvc.GenHashes()
	assert.Nil(t, err)
	assert.EqualValues(t, 1, hc)
}

func TestGenHasheshasErrorReturnsZero(t *testing.T) {
	teardown := setupTestCrawl()
	defer teardown()
	fi1 := domain.FileInfo{
		Path: "not-there",
	}
	crawlRepo.Store(fi1)
	hc, err := crawlSvc.GenHashes()
	assert.NotNil(t, err)
	assert.EqualValues(t, 0, hc)
}
