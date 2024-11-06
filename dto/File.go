// package dto defines the data structures used to exchange information
package dto

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/johannes-kuhfuss/mairlist-feeder/domain"
	"github.com/johannes-kuhfuss/mairlist-feeder/repositories"
)

// FileResp defines the data to be displayed in the file list
type FileResp struct {
	Path           string
	ModTime        string
	Duration       string
	StartTime      string
	EndTime        string
	InfoExtracted  string
	ScanTime       string
	FolderDate     string
	RuleMatched    string
	EventId        string
	EventIdLink    string
	EventLinkAvail bool
	CalCmsInfo     string
	TechMd         string
}

// FileCounts structure to list counts of file types
type FileCounts struct {
	TotalCount  int
	AudioCount  int
	StreamCount int
}

func formatTime(t1 time.Time) string {
	if t1.IsZero() {
		return "N/A"
	}
	return t1.Format("15:04")
}

func formatEventLink(id int) bool {
	return id != 0
}

// GetFiles retrives all files maintained in the repository and formats them for display purposes
func GetFiles(repo *repositories.DefaultFileRepository, CmsUrl string) (fileDta []FileResp) {
	if files := repo.GetAll(); files != nil {
		for _, file := range *files {
			dta := FileResp{
				Path:           file.Path,
				ModTime:        file.ModTime.Format("2006-01-02 15:04:05"),
				Duration:       strconv.FormatFloat(math.Round(file.Duration/60), 'f', 1, 64),
				InfoExtracted:  strconv.FormatBool(file.InfoExtracted),
				ScanTime:       file.ScanTime.Format("2006-01-02 15:04:05"),
				FolderDate:     file.FolderDate,
				RuleMatched:    file.RuleMatched,
				EventId:        strconv.Itoa(file.EventId),
				EventIdLink:    buildEventIdLink(CmsUrl, file.EventId),
				CalCmsInfo:     buildCalCmsInfo(file),
				TechMd:         buildTechMd(file),
				StartTime:      formatTime(file.StartTime),
				EndTime:        formatTime(file.EndTime),
				EventLinkAvail: formatEventLink(file.EventId),
			}
			fileDta = append(fileDta, dta)
		}
	}
	sort.SliceStable(fileDta, func(i, j int) bool {
		if strings.Compare(fileDta[i].StartTime, fileDta[j].StartTime) > 0 {
			return false
		} else {
			return true
		}
	})
	return
}

// buildCalCmsInfo formats information from calCms for display
func buildCalCmsInfo(file domain.FileInfo) (info string) {
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
		info = info + "\"" + file.CalCmsTitle + "\""
	} else {
		info = info + "None"
	}
	return
}

// buildTechMd formats information from ffprobe for display
func buildTechMd(file domain.FileInfo) (info string) {
	switch file.FileType {
	case "Audio":
		info = fmt.Sprintf("%v @ %vkbps", file.FormatName, file.BitRate)
	case "Stream":
		if file.StreamId != 0 {
			info = fmt.Sprintf("Stream %v with Id %v", file.StreamName, file.StreamId)
		} else {
			info = "N/A"
		}
	default:
		info = "N/A"
	}
	if file.Checksum != "" {
		info = info + " [" + file.Checksum + "]"
	}
	return
}

// buildEventIdLink returns a link to a calCms event
func buildEventIdLink(CmsUrl string, eventId int) string {
	// https://programm.coloradio.org/agenda/events.cgi?event_id=xxxxx
	idStr := strconv.Itoa(eventId)
	return CmsUrl + "?event_id=" + idStr
}
