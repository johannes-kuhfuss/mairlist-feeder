// Package repositories implements an in-memory store for representing the data of the files scanned
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
	AudioSize() int
	StreamSize() int
	GetByPath(string) *domain.FileInfo
	GetByEventId(int) *domain.FileList
	GetAll() *domain.FileList
	GetForHour(string) *domain.FileList
	Store(domain.FileInfo) error
	Delete(string) error
	SaveToDisk(string) error
	LoadFromDisk(string) error
	DeleteAllData()
	NewFiles() bool
}

type DefaultFileRepository struct {
	Cfg *config.AppConfig
}

var (
	fileList domain.SafeFileList
)

// NewFileRepository creates a new file repository. You need to pass in the configuration
func NewFileRepository(cfg *config.AppConfig) DefaultFileRepository {
	fileList.Files = make(map[string]domain.FileInfo)
	return DefaultFileRepository{
		Cfg: cfg,
	}
}

// Exists checks whether a file identified by its path exists in the repository
func (fr DefaultFileRepository) Exists(filePath string) bool {
	fileList.RLock()
	defer fileList.RUnlock()
	_, ok := fileList.Files[filePath]
	return ok
}

// Size returns the number of files stored in the repository
func (fr DefaultFileRepository) Size() int {
	fileList.RLock()
	defer fileList.RUnlock()
	return len(fileList.Files)
}

// SizeOfType returns the number of files of the specified fileType
func (fr DefaultFileRepository) sizeOfType(fileType string) (count int) {
	fileList.RLock()
	defer fileList.RUnlock()
	for _, f := range fileList.Files {
		if f.FileType == fileType {
			count++
		}
	}
	return
}

// AudioSize returns the number of audio files (as identified by their file extension) stored in the repository
func (fr DefaultFileRepository) AudioSize() int {
	return fr.sizeOfType("Audio")
}

// StreamSize returns the number of stream files (as identified by their file extension) stored in the repository
func (fr DefaultFileRepository) StreamSize() int {
	return fr.sizeOfType("Stream")
}

// GetByPath returns a file's information where the file is identified by its path. If no file matches, the methods returns nil
func (fr DefaultFileRepository) GetByPath(filePath string) *domain.FileInfo {
	var fi domain.FileInfo
	if !fr.Exists(filePath) {
		return nil
	}
	fileList.RLock()
	defer fileList.RUnlock()
	fi = fileList.Files[filePath]
	return &fi
}

// GetByEventId returns a file's information where the file is identified by its event id (from calCMS). If no file matches, the methods returns nil
func (fr DefaultFileRepository) GetByEventId(eventId int) *domain.FileList {
	var list domain.FileList
	if fr.Size() == 0 {
		return nil
	}
	fileList.RLock()
	defer fileList.RUnlock()
	for _, file := range fileList.Files {
		if file.EventId == eventId {
			list = append(list, file)
		}
	}
	if len(list) > 0 {
		return &list
	} else {
		return nil
	}
}

// GetAll returns all file data from the repository. Returns nil if repository is empty
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

// GetForHour returns all files' information that fall into a given start hour. If no files match, the methods returns nil
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

// Store stores a new file information entry into the repository
func (fr DefaultFileRepository) Store(fi domain.FileInfo) error {
	if fi.Path == "" {
		return errors.New("cannot add item with empty path to list")
	}
	fileList.Lock()
	defer fileList.Unlock()
	fileList.Files[fi.Path] = fi
	return nil
}

// Delete delete a file information entry from the repository, if it exists
func (fr DefaultFileRepository) Delete(filePath string) error {
	if !fr.Exists(filePath) {
		return fmt.Errorf("item with path %v does not exist", filePath)
	}
	fileList.Lock()
	defer fileList.Unlock()
	delete(fileList.Files, filePath)
	return nil
}

// SaveToDisk writes the repository's contents to a specified file on disk
func (fr DefaultFileRepository) SaveToDisk(fileName string) error {
	logger.Info("Saving files data to disk...")
	fileList.RLock()
	defer fileList.RUnlock()
	b, err := json.Marshal(fileList.Files)
	if err != nil {
		logger.Error("Error while converting file list to JSON: ", err)
		return err
	}
	if err := os.WriteFile(fileName, b, 0644); err != nil {
		logger.Error("Error while writing files data to disk: ", err)
		return err
	}
	logger.Infof("Done saving files data to disk (%v items).", len(fileList.Files))
	return nil
}

// LoadFromDisk loads file information stored on disk into memory
func (fr DefaultFileRepository) LoadFromDisk(fileName string) error {
	logger.Info("Reading files data from disk...")
	fileDta := make(map[string]domain.FileInfo)
	b, err := os.ReadFile(fileName)
	if err != nil {
		logger.Error("Error while reading files data from disk: ", err)
		return err
	}
	if err := json.Unmarshal(b, &fileDta); err != nil {
		logger.Error("Error while converting files data to json: ", err)
		return err
	}
	fileList.Lock()
	defer fileList.Unlock()
	fileList.Files = fileDta
	logger.Infof("Done reading files data from disk (%v items).", len(fileList.Files))
	return nil
}

// DeleteAllData removes all entries from the repository
func (fr DefaultFileRepository) DeleteAllData() {
	fileList.Lock()
	defer fileList.Unlock()
	fileList.Files = make(map[string]domain.FileInfo)
}

// NewFiles returns true, if there are file entries in the repository for which additional information hasn't been extracted
func (fr DefaultFileRepository) NewFiles() (newFiles bool) {
	if fr.Size() > 0 {
		allFiles := fr.GetAll()
		for _, file := range *allFiles {
			if !file.InfoExtracted {
				newFiles = true
				break
			}
		}
	}
	return
}
