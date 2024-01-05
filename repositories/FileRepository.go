package repositories

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/domain"
	"github.com/johannes-kuhfuss/services_utils/logger"
)

type FileRepository interface {
	Exists(string) bool
	Size() int
	GetFileData(string) domain.FileInfo
	StoreFileData(domain.FileInfo) error
	SaveToDisk(string)
	LoadFromDisk(string)
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

func (fr DefaultFileRepository) SaveToDisk(fileName string) {
	logger.Info("Saving files data to disk...")
	b, err := json.Marshal(fileList.files)
	if err != nil {
		logger.Error("Error while converting file list to JSON: ", err)
	}
	err = os.WriteFile(fileName, b, 0644)
	if err != nil {
		logger.Error("Error while writing files data to disk: ", err)
	}
	logger.Info(fmt.Sprintf("Done saving files data to disk (%v items).", len(fileList.files)))
}

func (fr DefaultFileRepository) LoadFromDisk(fileName string) {
	logger.Info("Reading files data from disk...")
	fileDta := make(map[string]domain.FileInfo)
	b, err := os.ReadFile(fileName)
	if err != nil {
		logger.Error("Error while reading files data from disk: ", err)
	}
	err = json.Unmarshal(b, &fileDta)
	if err != nil {
		logger.Error("Error while converting files data to json: ", err)
	}
	fileList.files = fileDta
	logger.Info(fmt.Sprintf("Done reading files data from disk (%v items).", len(fileList.files)))
}

func (fr DefaultFileRepository) DeleteAllData() {
	fileList.files = make(map[string]domain.FileInfo)
}
