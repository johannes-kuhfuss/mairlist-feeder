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
}

type FileList []FileInfo

// SafeFileList adds a mutex to allow thread-safe access of the file data entries
type SafeFileList struct {
	sync.RWMutex
	Files map[string]FileInfo
}
