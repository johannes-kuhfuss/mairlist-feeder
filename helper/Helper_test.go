package helper

import (
	"fmt"
	"path"
	"testing"
	"time"

	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/stretchr/testify/assert"
)

func Test_GetTodayFolder_test(t *testing.T) {
	folder := GetTodayFolder(true, "2024/01/31")
	assert.EqualValues(t, folder, "2024/01/31")
}

func Test_GetTodayFolder_Today(t *testing.T) {
	folder := GetTodayFolder(false, "")
	year := fmt.Sprintf("%d", time.Now().Year())
	month := fmt.Sprintf("%02d", time.Now().Month())
	day := fmt.Sprintf("%02d", time.Now().Day())
	testDate := path.Join(year, month, day)
	assert.EqualValues(t, folder, testDate)
}

func Test_TimeFromHourAndMinute_CorrectTime_ReturnsTime(t *testing.T) {
	t1 := TimeFromHourAndMinute(22, 22)
	t2 := time.Date(1, 1, 1, 22, 22, 0, 0, time.Local)
	assert.EqualValues(t, t2, t1)
}

func Test_TimeFromHourAndMinuteAndDate_CorrectTime_ReturnsTime(t *testing.T) {
	d := time.Date(2024, 2, 1, 0, 0, 0, 0, time.Local)
	t1 := TimeFromHourAndMinuteAndDate(22, 22, d)
	t2 := time.Date(2024, 2, 1, 22, 22, 0, 0, time.Local)
	assert.EqualValues(t, t2, t1)
}

func Test_IsAudioFile_NotIn_Returns_False(t *testing.T) {
	var cfg config.AppConfig
	config.InitConfig("", &cfg)
	path := "C:\\TEMP\\testfile.flac"
	isA := IsAudioFile(&cfg, path)
	assert.EqualValues(t, false, isA)
}

func Test_IsAudioFile_In_Returns_True(t *testing.T) {
	var cfg config.AppConfig
	config.InitConfig("", &cfg)
	path := "C:\\TEMP\\testfile.mp3"
	isA := IsAudioFile(&cfg, path)
	assert.EqualValues(t, true, isA)
}

func Test_IsStreamingFile_NotIn_Returns_False(t *testing.T) {
	var cfg config.AppConfig
	config.InitConfig("", &cfg)
	path := "C:\\TEMP\\testfile.xyz"
	isA := IsStreamingFile(&cfg, path)
	assert.EqualValues(t, false, isA)
}

func Test_IsStreamingFile_In_Returns_True(t *testing.T) {
	var cfg config.AppConfig
	config.InitConfig("", &cfg)
	path := "C:\\TEMP\\testfile.stream"
	isA := IsStreamingFile(&cfg, path)
	assert.EqualValues(t, true, isA)
}
