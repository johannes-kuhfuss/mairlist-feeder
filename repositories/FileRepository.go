package repositories

import (
	"errors"
	"strings"
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

func (fr DefaultFileRepository) Get(filePath string) *domain.FileInfo {
	var fi domain.FileInfo
	if !fr.Exists(filePath) {
		return nil
	}
	fileList.mu.Lock()
	fi = fileList.files[filePath]
	fileList.mu.Unlock()
	return &fi
}

func (fr DefaultFileRepository) GetAll() *domain.FileList {
	var list domain.FileList
	if fr.Size() == 0 {
		return nil
	}
	for _, file := range fileList.files {
		list = append(list, file)
	}
	return &list
}

func (fr DefaultFileRepository) GetForHour(hour string) *domain.FileList {
	var list domain.FileList
	if fr.Size() == 0 {
		return nil
	}
	for _, file := range fileList.files {
		if strings.HasPrefix(file.StartTime, hour) {
			list = append(list, file)
		}
	}
	if len(list) > 0 {
		return &list
	} else {
		return nil
	}

}

func (fr DefaultFileRepository) Store(fi domain.FileInfo) error {
	fileList.mu.Lock()
	if fi.Path == "" {
		return errors.New("cannot add item with empty path to list")
	}
	fileList.files[fi.Path] = fi
	fileList.mu.Unlock()
	return nil
}

func (fr DefaultFileRepository) Delete(filePath string) error {
	if !fr.Exists(filePath) {
		return errors.New("item does not exist")
	}
	fileList.mu.Lock()
	delete(fileList.files, filePath)
	fileList.mu.Unlock()
	return nil
}
