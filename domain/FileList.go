package domain

import (
	"os"
	"time"
)

type FileInfo struct {
	Path       string
	FileInfo   os.FileInfo
	Duration   float64
	StartTime  string
	FromCalCMS bool
	ScanTime   time.Time
}

type FileList []FileInfo
