package domain

import (
	"sync"
	"time"
)

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
}

type FileList []FileInfo

type SafeFileList struct {
	sync.RWMutex
	Files map[string]FileInfo
}
