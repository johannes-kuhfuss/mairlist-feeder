package helper

import (
	"fmt"
	"path"
	"time"

	"github.com/johannes-kuhfuss/services_utils/logger"
)

func GetTodayFolder(test bool, testDate string) string {
	var year, month, day string
	if test {
		return testDate
	}

	year = fmt.Sprintf("%d", time.Now().Year())
	month = fmt.Sprintf("%02d", time.Now().Month())
	day = fmt.Sprintf("%02d", time.Now().Day())

	// Detect 23:30 and advance by one day to start scanning next day's folder
	if (time.Now().Hour() == 23) && (time.Now().Minute() >= 30) {
		logger.Info("Scanning next day's folder now...")
		year = fmt.Sprintf("%d", time.Now().AddDate(0, 0, 1).Year())
		month = fmt.Sprintf("%02d", time.Now().AddDate(0, 0, 1).Month())
		day = fmt.Sprintf("%02d", time.Now().AddDate(0, 0, 1).Day())
	}

	return path.Join(year, month, day)
}

func TimeFromHourAndMinute(hour int, minute int) time.Time {
	t := time.Date(1, 1, 1, hour, minute, 0, 0, time.Local)
	return t
}

func TimeFromHourAndMinuteAndDate(hour int, minute int, fd time.Time) time.Time {
	t := time.Date(fd.Year(), fd.Month(), fd.Day(), hour, minute, 0, 0, time.Local)
	return t
}
