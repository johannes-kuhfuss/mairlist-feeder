package domain

import "os"

type FileInfo struct {
	Path     string
	FileInfo os.FileInfo
	Duration float64
	Slot     string
}

type FileList []FileInfo
