package helper

import (
	"fmt"
	"path"
	"time"
)

func GetTodayFolder(test bool) string {
	if !test {
		year := fmt.Sprintf("%d", time.Now().Year())
		month := fmt.Sprintf("%02d", time.Now().Month())
		day := fmt.Sprintf("%02d", time.Now().Day())

		return path.Join(year, month, day)
	} else {
		return path.Join("2023", "12")
	}
}
