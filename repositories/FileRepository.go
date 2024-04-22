package repositories

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/domain"
	"github.com/johannes-kuhfuss/mairlist-feeder/helper"
	"github.com/johannes-kuhfuss/services_utils/logger"
)

type FileRepository interface {
	Exists(string) bool
	Size() int
	GetFileData(string) domain.FileInfo
	StoreFileData(domain.FileInfo) error
	SaveToDisk(string)
	LoadFromDisk(string)
	DeleteAllData()
	NewFiles() bool
}

type DefaultFileRepository struct {
	Cfg *config.AppConfig
}

var (
	fileList domain.SafeFileList
)

func NewFileRepository(cfg *config.AppConfig) DefaultFileRepository {
	fileList.Files = make(map[string]domain.FileInfo)
	return DefaultFileRepository{
		Cfg: cfg,
	}
}

func (fr DefaultFileRepository) Exists(filePath string) bool {
	fileList.RLock()
	defer fileList.RUnlock()
	_, ok := fileList.Files[filePath]
	return ok
}

func (fr DefaultFileRepository) Size() int {
	fileList.RLock()
	defer fileList.RUnlock()
	return len(fileList.Files)
}

func (fr DefaultFileRepository) Get(filePath string) *domain.FileInfo {
	var fi domain.FileInfo
	if !fr.Exists(filePath) {
		return nil
	}
	fileList.RLock()
	defer fileList.RUnlock()
	fi = fileList.Files[filePath]
	return &fi
}

func (fr DefaultFileRepository) GetAll() *domain.FileList {
	var list domain.FileList
	if fr.Size() == 0 {
		return nil
	}
	fileList.RLock()
	defer fileList.RUnlock()
	for _, file := range fileList.Files {
		list = append(list, file)
	}
	return &list
}

func (fr DefaultFileRepository) GetForHour(hour string) *domain.FileList {
	var list domain.FileList
	if fr.Size() == 0 {
		return nil
	}
	folderDate := strings.Replace(helper.GetTodayFolder(fr.Cfg.Misc.TestCrawl, fr.Cfg.Misc.TestDate), "/", "-", -1)
	fileList.RLock()
	defer fileList.RUnlock()
	for _, file := range fileList.Files {
		hi, _ := strconv.Atoi(hour)
		if (!file.StartTime.IsZero()) && (file.StartTime.Hour() == hi) && (file.FolderDate == folderDate) {
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
	if fi.Path == "" {
		return errors.New("cannot add item with empty path to list")
	}
	fileList.Lock()
	defer fileList.Unlock()
	fileList.Files[fi.Path] = fi
	return nil
}

func (fr DefaultFileRepository) Delete(filePath string) error {
	if !fr.Exists(filePath) {
		return errors.New("item does not exist")
	}
	fileList.Lock()
	defer fileList.Unlock()
	delete(fileList.Files, filePath)
	return nil
}

func (fr DefaultFileRepository) SaveToDisk(fileName string) {
	logger.Info("Saving files data to disk...")
	fileList.RLock()
	defer fileList.RUnlock()
	b, err := json.Marshal(fileList.Files)
	if err != nil {
		logger.Error("Error while converting file list to JSON: ", err)
	}
	err = os.WriteFile(fileName, b, 0644)
	if err != nil {
		logger.Error("Error while writing files data to disk: ", err)
	}
	logger.Info(fmt.Sprintf("Done saving files data to disk (%v items).", len(fileList.Files)))
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
	fileList.Lock()
	defer fileList.Unlock()
	fileList.Files = fileDta
	logger.Info(fmt.Sprintf("Done reading files data from disk (%v items).", len(fileList.Files)))
}

func (fr DefaultFileRepository) DeleteAllData() {
	fileList.Lock()
	defer fileList.Unlock()
	fileList.Files = make(map[string]domain.FileInfo)
}

func (fr DefaultFileRepository) NewFiles() bool {
	newFiles := false
	if fr.Size() > 0 {
		allFiles := fr.GetAll()
		for _, file := range *allFiles {
			if !file.InfoExtracted {
				newFiles = true
				break
			}
		}
	}
	return newFiles
}
