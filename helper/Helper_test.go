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
