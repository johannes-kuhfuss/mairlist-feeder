package dto

import (
	"math"
	"strconv"

	"github.com/johannes-kuhfuss/mairlist-feeder/domain"
	"github.com/johannes-kuhfuss/mairlist-feeder/repositories"
)

type FileResp struct {
	Path          string
	ModTime       string
	Duration      string
	StartTime     string
	EndTime       string
	InfoExtracted string
	ScanTime      string
	FolderDate    string
	RuleMatched   string
	EventId       string
	CalCmsInfo    string
	TechMd        string
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
				InfoExtracted: strconv.FormatBool(file.InfoExtracted),
				ScanTime:      file.ScanTime.Format("2006-01-02 15:04:05"),
				FolderDate:    file.FolderDate,
				RuleMatched:   file.RuleMatched,
				EventId:       strconv.Itoa(file.EventId),
				CalCmsInfo:    buildCalCmsInfo(file),
				TechMd:        buildTechMd(file),
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

func buildCalCmsInfo(file domain.FileInfo) string {
	var info string
	if file.FromCalCMS {
		info = "Yes, "
	} else {
		info = "No, "
	}
	if file.CalCmsInfoExtracted {
		info = info + "Yes, "
	} else {
		info = info + "No, "
	}
	if file.CalCmsTitle != "" {
		info = info + file.CalCmsTitle
	} else {
		info = info + "None"
	}
	return info
}

func buildTechMd(file domain.FileInfo) string {
	info := strconv.FormatInt(file.BitRate, 10) + ", " + file.FormatName
	return info
}
