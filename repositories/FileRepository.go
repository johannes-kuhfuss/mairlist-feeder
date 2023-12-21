package repositories

import (
	"sync"

	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/domain"
)

type FileRepository interface {
	Exists(string) bool
	Size() int
	GetFileData(string) domain.FileInfo
	StoreFileData(domain.FileInfo) error
}

type DefaultFileRepository struct {
	Cfg *config.AppConfig
}

type safeFileList struct {
	mu    sync.Mutex
	files map[string]domain.FileInfo
}

var (
	fileList safeFileList
)

func NewFileRepository(cfg *config.AppConfig) DefaultFileRepository {
	fileList.files = make(map[string]domain.FileInfo)
	return DefaultFileRepository{
		Cfg: cfg,
	}
}

func (fr DefaultFileRepository) Exists(filePath string) bool {
	_, ok := fileList.files[filePath]
	return ok
}

func (fr DefaultFileRepository) Size() int {
	return len(fileList.files)
}

func (fr DefaultFileRepository) GetFileData(filePath string) *domain.FileInfo {
	var fi domain.FileInfo
	if !fr.Exists(filePath) {
		return nil
	}
	fileList.mu.Lock()
	fi = fileList.files[filePath]
	fileList.mu.Unlock()
	return &fi
}

func (fr DefaultFileRepository) Store(fi domain.FileInfo) error {
	fileList.mu.Lock()
	fileList.files[fi.Path] = fi
	fileList.mu.Unlock()
	return nil
}
