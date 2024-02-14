package dto

import (
	"math"
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
	EventId       string
	CalCmsInfo    string
	CalCmsTitle   string
}

func GetFiles(repo *repositories.DefaultFileRepository) []FileResp {
	var (
		fileDta []FileResp
	)
	files := repo.GetAll()
	if files != nil {
		for _, file := range *files {
			dta := FileResp{
				Path:          file.Path,
				ModTime:       file.ModTime.Format("2006-01-02 15:04:05"),
				Duration:      strconv.FormatFloat(math.Round(file.Duration/60), 'f', 1, 64),
				FromCalCMS:    strconv.FormatBool(file.FromCalCMS),
				InfoExtracted: strconv.FormatBool(file.InfoExtracted),
				ScanTime:      file.ScanTime.Format("2006-01-02 15:04:05"),
				FolderDate:    file.FolderDate,
				RuleMatched:   file.RuleMatched,
				EventId:       strconv.Itoa(file.EventId),
				CalCmsInfo:    strconv.FormatBool(file.CalCmsInfoExtracted),
				CalCmsTitle:   file.CalCmsTitle,
			}
			if file.StartTime.IsZero() {
				dta.StartTime = "N/A"
			} else {
				dta.StartTime = file.StartTime.Format("15:04")
			}
			if file.EndTime.IsZero() {
				dta.EndTime = "N/A"
			} else {
				dta.EndTime = file.EndTime.Format("15:04")
			}
			fileDta = append(fileDta, dta)
		}
	}
	return fileDta
}
