package helper

import (
	"fmt"
	"path"
	"testing"
	"time"

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
