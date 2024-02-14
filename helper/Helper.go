package helper

import (
	"fmt"
	"path"
	"time"
)

func GetTodayFolder(test bool, testDate string) string {
	if test {
		return testDate
	}

	year := fmt.Sprintf("%d", time.Now().Year())
	month := fmt.Sprintf("%02d", time.Now().Month())
	day := fmt.Sprintf("%02d", time.Now().Day())

	return path.Join(year, month, day)
}

func TimeFromHourAndMinute(hour int, minute int) time.Time {
	t := time.Date(1, 1, 1, hour, minute, 0, 0, time.Local)
	return t
}
