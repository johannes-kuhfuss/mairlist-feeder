package repositories

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/domain"
	"github.com/johannes-kuhfuss/mairlist-feeder/helper"
	"github.com/stretchr/testify/assert"
)

var (
	cfg  config.AppConfig
	repo DefaultFileRepository
)

const (
	testFile       = "test.dta"
	folderDateDash = "2024-09-17"
)

func setupTest() {
	repo = NewFileRepository(&cfg)
}

func todayFolderDate() time.Time {
	return domain.MustParseFolderDate(strings.Replace(helper.GetTodayFolder(false, ""), "/", "-", -1))
}

func TestNewFileRepositoryCreatesEmptyList(t *testing.T) {
	setupTest()
	assert.EqualValues(t, 0, repo.Size())
}

func TestNewFileRepositoryInstancesDoNotShareData(t *testing.T) {
	repo1 := NewFileRepository(&cfg)
	repo2 := NewFileRepository(&cfg)

	err := repo1.Store(domain.FileInfo{Path: "A"})

	assert.Nil(t, err)
	assert.EqualValues(t, 1, repo1.Size())
	assert.EqualValues(t, 0, repo2.Size())
	assert.Nil(t, repo2.GetByPath("A"))
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
		Duration: time.Second,
	}
	err := repo.Store(fi)
	res := repo.GetByPath("A")
	assert.Nil(t, err)
	assert.NotNil(t, res)
	assert.EqualValues(t, "A", res.Path)
	assert.EqualValues(t, time.Second, res.Duration)
}

func TestGetAllReturnsAllElements(t *testing.T) {
	setupTest()
	fi1 := domain.FileInfo{
		Path:     "A",
		Duration: 2 * time.Second,
	}
	fi2 := domain.FileInfo{
		Path:     "B",
		Duration: 2 * time.Second,
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
	assert.EqualValues(t, 2*time.Second, el1.Duration)
	assert.EqualValues(t, 2*time.Second, el2.Duration)
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
		Duration: time.Second,
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
	res := repo.GetByHour("13", false)
	assert.Nil(t, res)
}

func TestGetForHourNoMatchReturnsNil(t *testing.T) {
	setupTest()
	fi := domain.FileInfo{
		Path:       "A",
		StartTime:  helper.TimeFromHourAndMinute(12, 0),
		FolderDate: todayFolderDate(),
	}
	repo.Store(fi)
	res := repo.GetByHour("13", false)
	assert.Nil(t, res)
}

func TestGetForHourOneMatchReturnsElement(t *testing.T) {
	setupTest()
	fi := domain.FileInfo{
		Path:       "A",
		StartTime:  helper.TimeFromHourAndMinute(12, 0),
		FolderDate: todayFolderDate(),
	}
	repo.Store(fi)
	res := repo.GetByHour("12", false)
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
		FolderDate: todayFolderDate(),
	}
	fi2 := domain.FileInfo{
		Path:       "B",
		StartTime:  helper.TimeFromHourAndMinute(12, 0),
		FolderDate: todayFolderDate(),
	}
	fi3 := domain.FileInfo{
		Path:       "C",
		StartTime:  helper.TimeFromHourAndMinute(12, 30),
		FolderDate: todayFolderDate(),
	}
	repo.Store(fi1)
	repo.Store(fi2)
	repo.Store(fi3)
	res := repo.GetByHour("12", false)
	assert.NotNil(t, *res)
	assert.EqualValues(t, 2, len(*res))
}

func TestGetForHourOnlyLiveReturnsNil(t *testing.T) {
	setupTest()
	fi1 := domain.FileInfo{
		Path:        "A",
		StartTime:   helper.TimeFromHourAndMinute(12, 0),
		FolderDate:  todayFolderDate(),
		EventIsLive: true,
	}
	fi2 := domain.FileInfo{
		Path:        "B",
		StartTime:   helper.TimeFromHourAndMinute(12, 0),
		FolderDate:  todayFolderDate(),
		EventIsLive: true,
	}
	repo.Store(fi1)
	repo.Store(fi2)
	res := repo.GetByHour("12", false)
	assert.Nil(t, res)
}

func TestGetForHourLiveCheck(t *testing.T) {
	setupTest()
	fi1 := domain.FileInfo{
		Path:        "A",
		StartTime:   helper.TimeFromHourAndMinute(12, 0),
		FolderDate:  todayFolderDate(),
		EventIsLive: false,
	}
	fi2 := domain.FileInfo{
		Path:        "B",
		StartTime:   helper.TimeFromHourAndMinute(12, 0),
		FolderDate:  todayFolderDate(),
		EventIsLive: true,
	}
	repo.Store(fi1)
	repo.Store(fi2)
	res1 := repo.GetByHour("12", false)
	assert.NotNil(t, *res1)
	assert.EqualValues(t, 1, len(*res1))
	assert.EqualValues(t, "A", (*res1)[0].Path)
	res2 := repo.GetByHour("12", true)
	assert.NotNil(t, *res2)
	assert.EqualValues(t, 2, len(*res2))
}

func TestGetByDateAndHourOnlyReturnsRequestedDate(t *testing.T) {
	setupTest()
	folderDate := domain.MustParseFolderDate("2024-09-17")
	nextDate := domain.MustParseFolderDate("2024-09-18")
	fi1 := domain.FileInfo{
		Path:       "A",
		StartTime:  helper.TimeFromHourAndMinuteAndDate(12, 0, folderDate),
		FolderDate: folderDate,
	}
	fi2 := domain.FileInfo{
		Path:       "B",
		StartTime:  helper.TimeFromHourAndMinuteAndDate(12, 0, nextDate),
		FolderDate: nextDate,
	}
	repo.Store(fi1)
	repo.Store(fi2)

	res := repo.GetByDateAndHour(nextDate, "12", false)

	assert.NotNil(t, res)
	assert.EqualValues(t, 1, len(*res))
	assert.EqualValues(t, "B", (*res)[0].Path)
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
	err1 := repo.SaveToDisk(testFile)
	repo.DeleteAllData()
	sizeBefore := repo.Size()
	err2 := repo.LoadFromDisk(testFile)
	sizeAfter := repo.Size()
	assert.Nil(t, err1)
	assert.Nil(t, err2)
	assert.EqualValues(t, 0, sizeBefore)
	assert.EqualValues(t, 3, sizeAfter)
	os.Remove(testFile)
}

func TestSaveToDiskReplacesExistingFileAndCleansTempFile(t *testing.T) {
	setupTest()
	dir := t.TempDir()
	fileName := filepath.Join(dir, "files.dta")
	os.WriteFile(fileName, []byte("old data"), 0644)
	fi := domain.FileInfo{
		Path:      "A",
		StartTime: helper.TimeFromHourAndMinute(11, 0),
	}
	repo.Store(fi)

	err := repo.SaveToDisk(fileName)
	matches, globErr := filepath.Glob(filepath.Join(dir, "files.dta.*.tmp"))
	data, readErr := os.ReadFile(fileName)

	assert.Nil(t, err)
	assert.Nil(t, globErr)
	assert.Empty(t, matches)
	assert.Nil(t, readErr)
	assert.Contains(t, string(data), "\"A\"")
	assert.NotContains(t, string(data), "old data")
}

func TestLoadFromDiskNoFileReturnsError(t *testing.T) {
	setupTest()
	err := repo.LoadFromDisk("no.file")
	assert.NotNil(t, err)
	assert.True(t, os.IsNotExist(err))
}

func TestLoadFromDiskWrongDataReturnsError(t *testing.T) {
	setupTest()
	os.WriteFile(testFile, []byte("someBogusData"), 0666)
	err := repo.LoadFromDisk(testFile)
	assert.NotNil(t, err)
	assert.EqualValues(t, "invalid character 's' looking for beginning of value", err.Error())
	os.Remove(testFile)
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
		FileType:  domain.FileTypeAudio,
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
		FileType:  domain.FileTypeStream,
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
	assert.ElementsMatch(t, []string{"A", "B"}, []string{(*res)[0].Path, (*res)[1].Path})
}

func TestGetByEventIdAndDateOnlyReturnsRequestedDate(t *testing.T) {
	setupTest()
	folderDate := domain.MustParseFolderDate("2024-09-17")
	nextDate := domain.MustParseFolderDate("2024-09-18")
	fi1 := domain.FileInfo{
		Path:       "A",
		EventId:    1,
		FolderDate: folderDate,
	}
	fi2 := domain.FileInfo{
		Path:       "B",
		EventId:    1,
		FolderDate: nextDate,
	}
	repo.Store(fi1)
	repo.Store(fi2)

	res := repo.GetByEventIdAndDate(1, nextDate)

	assert.NotNil(t, res)
	assert.EqualValues(t, 1, len(*res))
	assert.EqualValues(t, "B", (*res)[0].Path)
}

func TestGetByDateEmptyReturnsNil(t *testing.T) {
	setupTest()
	res := repo.GetByDate(time.Time{})
	assert.Nil(t, res)
}

func TestGetByDateNoMatchReturnsNil(t *testing.T) {
	setupTest()
	fi1 := domain.FileInfo{
		Path:       "A",
		FolderDate: domain.MustParseFolderDate(folderDateDash),
	}
	repo.Store(fi1)
	res := repo.GetByDate(domain.MustParseFolderDate("2024-09-18"))
	assert.Nil(t, res)
}

func TestGetByDateOneMatchReturnsMatch(t *testing.T) {
	setupTest()
	fi1 := domain.FileInfo{
		Path:       "A",
		FolderDate: domain.MustParseFolderDate(folderDateDash),
	}
	repo.Store(fi1)
	res := repo.GetByDate(domain.MustParseFolderDate(folderDateDash))
	assert.NotNil(t, res)
	assert.EqualValues(t, "A", (*res)[0].Path)
}

func TestGetByDateTwoMatchesReturnsMatches(t *testing.T) {
	setupTest()
	fi1 := domain.FileInfo{
		Path:       "A",
		FolderDate: domain.MustParseFolderDate(folderDateDash),
	}
	fi2 := domain.FileInfo{
		Path:       "B",
		FolderDate: domain.MustParseFolderDate(folderDateDash),
	}
	repo.Store(fi1)
	repo.Store(fi2)
	res := repo.GetByDate(domain.MustParseFolderDate(folderDateDash))
	assert.NotNil(t, res)
	assert.EqualValues(t, 2, len(*res))
}

func TestMergeFileListNil(t *testing.T) {
	flm := mergeFileList(nil, nil)
	assert.EqualValues(t, 0, len(flm))
}

func TestMergeFileListEmpty(t *testing.T) {
	fl1 := domain.FileList{}
	fl2 := domain.FileList{}
	flm := mergeFileList(&fl1, &fl2)
	assert.EqualValues(t, 0, len(flm))
}

func TestMergeFileListTwoDifferentElements(t *testing.T) {
	fi1 := domain.FileInfo{Path: "A"}
	fi2 := domain.FileInfo{Path: "B"}
	fl1 := domain.FileList{fi1}
	fl2 := domain.FileList{fi2}
	flm := mergeFileList(&fl1, &fl2)
	assert.EqualValues(t, 2, len(flm))
}

func TestMergeFileListTwoEqualElements(t *testing.T) {
	fi1 := domain.FileInfo{Path: "A"}
	fi2 := domain.FileInfo{Path: "A"}
	fi3 := domain.FileInfo{Path: "B"}
	fl1 := domain.FileList{fi1}
	fl2 := domain.FileList{fi2, fi3}
	flm := mergeFileList(&fl1, &fl2)
	assert.EqualValues(t, 2, len(flm))
}
