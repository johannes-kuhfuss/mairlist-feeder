package dto

import (
	"strconv"

	"github.com/johannes-kuhfuss/mairlist-feeder/repositories"
)

type FileResp struct {
	Path          string
	ModTime       string
	Duration      string
	StartTime     string
	EndTime       string
	FromCalCMS    string
	InfoExtracted string
	ScanTime      string
	FolderDate    string
	RuleMatched   string
}

func GetFiles(repo *repositories.DefaultFileRepository) []FileResp {
	var (
		fileDta []FileResp
	)
	files := repo.GetAll()
	for _, file := range *files {
		dta := FileResp{
			Path:          file.Path,
			ModTime:       file.ModTime.Format("2006-01-02 15:04:05 -0700"),
			Duration:      strconv.FormatFloat(file.Duration, 'f', 1, 64),
			StartTime:     file.StartTime,
			EndTime:       file.EndTime,
			FromCalCMS:    strconv.FormatBool(file.FromCalCMS),
			InfoExtracted: strconv.FormatBool(file.InfoExtracted),
			ScanTime:      file.ScanTime.Format("2006-01-02 15:04:05 -0700"),
			FolderDate:    file.FolderDate,
			RuleMatched:   file.RuleMatched,
		}
		fileDta = append(fileDta, dta)
	}
	return fileDta
}
