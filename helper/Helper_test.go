package helper

import (
	"fmt"
	"path"
	"testing"
	"time"

	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/stretchr/testify/assert"
)

func TestGetTodayFoldertest(t *testing.T) {
	folder := GetTodayFolder(true, "2024/01/31")
	assert.EqualValues(t, folder, "2024/01/31")
}

func TestGetTodayFolderToday(t *testing.T) {
	folder := GetTodayFolder(false, "")
	year := fmt.Sprintf("%d", time.Now().Year())
	month := fmt.Sprintf("%02d", time.Now().Month())
	day := fmt.Sprintf("%02d", time.Now().Day())
	testDate := path.Join(year, month, day)
	assert.EqualValues(t, folder, testDate)
}

func TestTimeFromHourAndMinuteCorrectTimeReturnsTime(t *testing.T) {
	t1 := TimeFromHourAndMinute(22, 22)
	t2 := time.Date(1, 1, 1, 22, 22, 0, 0, time.Local)
	assert.EqualValues(t, t2, t1)
}

func TestTimeFromHourAndMinuteAndDateCorrectTimeReturnsTime(t *testing.T) {
	d := time.Date(2024, 2, 1, 0, 0, 0, 0, time.Local)
	t1 := TimeFromHourAndMinuteAndDate(22, 22, d)
	t2 := time.Date(2024, 2, 1, 22, 22, 0, 0, time.Local)
	assert.EqualValues(t, t2, t1)
}

func TestIsAudioFileNotInReturnsFalse(t *testing.T) {
	var cfg config.AppConfig
	config.InitConfig("", &cfg)
	path := "C:\\TEMP\\testfile.flac"
	isA := IsAudioFile(&cfg, path)
	assert.EqualValues(t, false, isA)
}

func TestIsAudioFileInReturnsTrue(t *testing.T) {
	var cfg config.AppConfig
	config.InitConfig("", &cfg)
	path := "C:\\TEMP\\testfile.mp3"
	isA := IsAudioFile(&cfg, path)
	assert.EqualValues(t, true, isA)
}

func TestIsStreamingFileNotInReturnsFalse(t *testing.T) {
	var cfg config.AppConfig
	config.InitConfig("", &cfg)
	path := "C:\\TEMP\\testfile.xyz"
	isA := IsStreamingFile(&cfg, path)
	assert.EqualValues(t, false, isA)
}

func TestIsStreamingFileInReturnsTrue(t *testing.T) {
	var cfg config.AppConfig
	config.InitConfig("", &cfg)
	path := "C:\\TEMP\\testfile.stream"
	isA := IsStreamingFile(&cfg, path)
	assert.EqualValues(t, true, isA)
}
