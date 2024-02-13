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

func Test_TimeFromHourAndMinute_WrongHour_ReturnsError(t *testing.T) {
	t1, err := TimeFromHourAndMinute(25, 25)
	assert.Nil(t, t1)
	assert.NotNil(t, err)
	assert.EqualValues(t, "hour must be between 0 and 23, minute must be between 0 and 59", err.Error())
}

func Test_TimeFromHourAndMinute_WrongMinute_ReturnsError(t *testing.T) {
	t1, err := TimeFromHourAndMinute(22, 72)
	assert.Nil(t, t1)
	assert.NotNil(t, err)
	assert.EqualValues(t, "hour must be between 0 and 23, minute must be between 0 and 59", err.Error())
}

func Test_TimeFromHourAndMinute_CorrectTime_ReturnsTime(t *testing.T) {
	t1, err := TimeFromHourAndMinute(22, 22)
	t2 := time.Date(1, 1, 1, 22, 22, 0, 0, time.Local)
	assert.NotNil(t, t1)
	assert.Nil(t, err)
	assert.EqualValues(t, &t2, t1)
}
