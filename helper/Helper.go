// Package helper implements a few functions that assist processing in the rest of the packages
package helper

import (
	"fmt"
	"path"
	"path/filepath"
	"time"

	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/services_utils/misc"
)

// GetTodayFolder returns today's date in folder syntax (YYYY/MM/DD).
// 30 minutes before the date rolls over, GetTodayFolder returns the next day's date.
// For testing, you can pass in a test date which is then returns to the caller
func GetTodayFolder(test bool, testDate string) string {
	var year, month, day string
	if test {
		return testDate
	}

	// Detect 23:30 and advance by one day to start returning next day's folder
	if (time.Now().Hour() == 23) && (time.Now().Minute() >= 30) {
		year = fmt.Sprintf("%d", time.Now().AddDate(0, 0, 1).Year())
		month = fmt.Sprintf("%02d", time.Now().AddDate(0, 0, 1).Month())
		day = fmt.Sprintf("%02d", time.Now().AddDate(0, 0, 1).Day())
	} else {
		year = fmt.Sprintf("%d", time.Now().Year())
		month = fmt.Sprintf("%02d", time.Now().Month())
		day = fmt.Sprintf("%02d", time.Now().Day())
	}

	return path.Join(year, month, day)
}

// TimeFromHourAndMinute generates a time.Time{} from an hour and minute value
func TimeFromHourAndMinute(hour int, minute int) time.Time {
	return time.Date(1, 1, 1, hour, minute, 0, 0, time.Local)
}

// TimeFromHourAndMinute generates a time.Time{} from an hour and minute and date value
func TimeFromHourAndMinuteAndDate(hour int, minute int, fd time.Time) time.Time {
	return time.Date(fd.Year(), fd.Month(), fd.Day(), hour, minute, 0, 0, time.Local)
}

// IsAudioFile returns true, if a file's extension is in the configured audio file extensions
func IsAudioFile(cfg *config.AppConfig, path string) bool {
	return misc.SliceContainsStringCI(cfg.Crawl.AudioFileExtensions, filepath.Ext(path))
}

// IsStreamingFile returns true, if a file's extension is in the configured streaming file extensions
func IsStreamingFile(cfg *config.AppConfig, path string) bool {
	return misc.SliceContainsStringCI(cfg.Crawl.StreamingFileExtensions, filepath.Ext(path))
}
