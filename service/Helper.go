package service

import (
	"path"
)

func getTodayFolder() string {
	/*
		year := fmt.Sprintf("%d", time.Now().Year())
		month := fmt.Sprintf("%02d", time.Now().Month())
		day := fmt.Sprintf("%02d", time.Now().Day())

		return path.Join(year, month, day)
	*/
	return path.Join("2023", "12", "22")
}