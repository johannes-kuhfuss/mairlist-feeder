package repositories

import (
	"sync"

	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/domain"
)

type FileRepository interface {
	IsPresent(string) bool
	Size() int
	GetFileData(string) domain.FileInfo
	StoreFileData(domain.FileInfo) error
}

type DefaultFileRepository struct {
	Cfg *config.AppConfig
}

type safeFileList struct {
	mu    sync.Mutex
	files []domain.FileInfo
}

var (
	fileList safeFileList
)

func NewFileRepository(cfg *config.AppConfig) DefaultFileRepository {
	return DefaultFileRepository{
		Cfg: cfg,
	}
}

func (fr DefaultFileRepository) IsPresent(filePath string) bool {
	return false
}

func (fr DefaultFileRepository) Size() int {
	return len(fileList.files)
}

func (fr DefaultFileRepository) GetFileData(filePath string) domain.FileInfo {
	var fi domain.FileInfo
	return fi
}

func (fr DefaultFileRepository) Store(fi domain.FileInfo) error {
	fileList.mu.Lock()
	fileList.files = append(fileList.files, fi)
	fileList.mu.Unlock()
	return nil
}
