package helper

import (
	"errors"
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

func TimeFromHourAndMinute(hour int, minute int) (*time.Time, error) {
	if (hour < 0) || (hour > 23) || (minute < 0) || (minute > 59) {
		return nil, errors.New("hour must be between 0 and 23, minute must be between 0 and 59")
	}
	t := time.Date(1, 1, 1, hour, minute, 0, 0, time.Local)
	return &t, nil
}
