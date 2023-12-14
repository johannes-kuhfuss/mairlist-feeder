package domain

import "os"

type FileInfo struct {
	FilePath string
	FileInfo os.FileInfo
}

type FileList []FileInfo
