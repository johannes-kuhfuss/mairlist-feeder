package helper

import (
	"path"
)

func GetTodayFolder() string {
	/*
		year := fmt.Sprintf("%d", time.Now().Year())
		month := fmt.Sprintf("%02d", time.Now().Month())
		day := fmt.Sprintf("%02d", time.Now().Day())

		return path.Join(year, month, day)
	*/
	return path.Join("2024", "01", "07")
}
