package domain

import (
	"os"
	"time"
)

type FileInfo struct {
	Path          string
	FileInfo      os.FileInfo
	Duration      float64
	StartTime     string
	FromCalCMS    bool
	InfoExtracted bool
	ScanTime      time.Time
	FolderDate    string
}

type FileList []FileInfo