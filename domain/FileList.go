package domain

import (
	"time"
)

type FileInfo struct {
	Path          string
	ModTime       time.Time
	Duration      float64
	StartTime     string
	EndTime       string
	FromCalCMS    bool
	InfoExtracted bool
	ScanTime      time.Time
	FolderDate    string
	RuleMatched   string
}

type FileList []FileInfo
