// Package helper implements a few functions that assist processing in the rest of the packages
package helper

import (
	"fmt"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/johannes-kuhfuss/mairlist-feeder/config"
)

// GetTodayFolder returns today's date in folder syntax (YYYY/MM/DD).
// For testing, you can pass in a test date which is then returned to the caller.
func GetTodayFolder(test bool, testDate string) string {
	return FolderForDate(DateForFolder(test, testDate, 0))
}

// DateForFolder returns the base folder date plus an offset in days.
func DateForFolder(test bool, testDate string, offsetDays int) time.Time {
	baseDate := time.Now()
	if test {
		parsedDate, err := time.ParseInLocation("2006/01/02", testDate, time.Local)
		if err == nil {
			baseDate = parsedDate
		}
	}
	return time.Date(baseDate.Year(), baseDate.Month(), baseDate.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, offsetDays)
}

// GetCrawlDates returns the folder dates that should be crawled.
func GetCrawlDates(test bool, testDate string) []time.Time {
	return []time.Time{
		DateForFolder(test, testDate, 0),
		DateForFolder(test, testDate, 1),
	}
}

// FolderForDate formats a date as YYYY/MM/DD for the crawl folder layout.
func FolderForDate(date time.Time) string {
	year := fmt.Sprintf("%d", date.Year())
	month := fmt.Sprintf("%02d", date.Month())
	day := fmt.Sprintf("%02d", date.Day())
	return path.Join(year, month, day)
}

// TimeFromHourAndMinute generates a time.Time{} from an hour and minute value
func TimeFromHourAndMinute(hour, minute int) time.Time {
	return time.Date(1, 1, 1, hour, minute, 0, 0, time.Local)
}

// TimeFromHourAndMinute generates a time.Time{} from an hour and minute and date value
func TimeFromHourAndMinuteAndDate(hour, minute int, fd time.Time) time.Time {
	return time.Date(fd.Year(), fd.Month(), fd.Day(), hour, minute, 0, 0, time.Local)
}

// IsAudioFile returns true, if a file's extension is in the configured audio file extensions
func IsAudioFile(cfg *config.AppConfig, path string) bool {
	return slices.ContainsFunc(cfg.Crawl.AudioFileExtensions, func(s string) bool { return strings.EqualFold(s, filepath.Ext(path)) })
	//return misc.SliceContainsStringCI(cfg.Crawl.AudioFileExtensions, filepath.Ext(path))
}

// IsStreamingFile returns true, if a file's extension is in the configured streaming file extensions
func IsStreamingFile(cfg *config.AppConfig, path string) bool {
	return slices.ContainsFunc(cfg.Crawl.StreamingFileExtensions, func(s string) bool { return strings.EqualFold(s, filepath.Ext(path)) })
	//return misc.SliceContainsStringCI(cfg.Crawl.StreamingFileExtensions, filepath.Ext(path))
}
