// package domain defines the core data structures
package domain

import (
	"sync"
	"time"
)

// FileInfo defines the information maintained per file entry
type FileInfo struct {
	Path                string
	ModTime             time.Time
	Duration            float64
	StartTime           time.Time
	EndTime             time.Time
	FromCalCMS          bool
	InfoExtracted       bool
	ScanTime            time.Time
	FolderDate          string
	RuleMatched         string
	EventId             int
	CalCmsTitle         string
	CalCmsInfoExtracted bool
	BitRate             int64
	FormatName          string
	SlotLength          float64
	FileType            string
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
func (fl FileList) Less(i int, j int) bool {
	return fl[i].StartTime.Before(fl[j].StartTime)
}

// Swap implements sort.Interface.
func (fl FileList) Swap(i int, j int) {
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
