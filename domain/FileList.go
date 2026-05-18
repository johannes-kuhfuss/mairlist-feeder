// package domain defines the core data structures
package domain

import (
	"fmt"
	"sync"
	"time"
)

const FolderDateLayout = "2006-01-02"

type FileType string

const (
	FileTypeAudio  FileType = "Audio"
	FileTypeStream FileType = "Stream"
)

// FileInfo defines the information maintained per file entry
type FileInfo struct {
	Path                string
	ModTime             time.Time
	Duration            time.Duration
	StartTime           time.Time
	EndTime             time.Time
	FromCalCMS          bool
	InfoExtracted       bool
	ScanTime            time.Time
	FolderDate          time.Time
	RuleMatched         string
	EventId             int
	CalCmsTitle         string
	CalCmsInfoExtracted bool
	BitRate             int64
	FormatName          string
	SlotLength          time.Duration
	FileType            FileType
	StreamId            int
	StreamName          string
	Checksum            string
	EventIsLive         bool
}

type FileList []FileInfo

// Len implements sort.Interface.
func (fl FileList) Len() int {
	return len(fl)
}

// Less implements sort.Interface.
func (fl FileList) Less(i, j int) bool {
	return fl[i].StartTime.Before(fl[j].StartTime)
}

// Swap implements sort.Interface.
func (fl FileList) Swap(i, j int) {
	fl[i], fl[j] = fl[j], fl[i]
}

// SafeFileList adds a mutex to allow thread-safe access of the file data entries
type SafeFileList struct {
	sync.RWMutex
	Files map[string]FileInfo
}

func (fl FileList) ContainsPath(path string) bool {
	for _, e := range fl {
		if e.Path == path {
			return true
		}
	}
	return false
}

func ParseFolderDate(value string) (time.Time, error) {
	date, err := time.ParseInLocation(FolderDateLayout, value, time.Local)
	if err != nil {
		return time.Time{}, err
	}
	return NormalizeDate(date), nil
}

func MustParseFolderDate(value string) time.Time {
	date, err := ParseFolderDate(value)
	if err != nil {
		panic(fmt.Sprintf("invalid folder date %q: %v", value, err))
	}
	return date
}

func NormalizeDate(value time.Time) time.Time {
	if value.IsZero() {
		return time.Time{}
	}
	local := value.In(time.Local)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.Local)
}

func FormatFolderDate(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return NormalizeDate(value).Format(FolderDateLayout)
}
