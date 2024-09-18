package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_isYesterdayOrOlder_IsOlder_ReturnsTrue(t *testing.T) {
	testDate := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
	testDateStr := testDate.Format("2006-01-02")
	b := isYesterdayOrOlder(testDateStr)
	assert.EqualValues(t, true, b)
}

func Test_isYesterdayOrOlder_IsWayOlder_ReturnsTrue(t *testing.T) {
	testDate := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC).AddDate(0, -1, 0)
	testDateStr := testDate.Format("2006-01-02")
	b := isYesterdayOrOlder(testDateStr)
	assert.EqualValues(t, true, b)
}

func Test_isYesterdayOrOlder_IsNotOlder_ReturnsFalse(t *testing.T) {
	testDate := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC)
	testDateStr := testDate.Format("2006-01-02")
	b := isYesterdayOrOlder(testDateStr)
	assert.EqualValues(t, false, b)
}
