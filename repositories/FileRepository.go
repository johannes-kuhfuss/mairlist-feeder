// Package repositories implements an in-memory store for representing the data of the files scanned
package repositories

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

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
	GetByEventId(int) domain.FileList
	GetByEventIdAndDate(int, time.Time) domain.FileList
	GetAll() domain.FileList
	GetByDate(time.Time) domain.FileList
	GetByHour(string, bool) domain.FileList
	GetByDateAndHour(time.Time, string, bool) domain.FileList
	GetByIdAndDateAndHour(int, time.Time, string, bool) domain.FileList
	Store(domain.FileInfo) error
	Delete(string) error
	SaveToDisk(string) error
	LoadFromDisk(string) error
	DeleteAllData()
	NewFiles() bool
}

type DefaultFileRepository struct {
	Cfg   *config.AppConfig
	files *domain.SafeFileList
}

// NewFileRepository creates a new file repository. You need to pass in the configuration
func NewFileRepository(cfg *config.AppConfig) DefaultFileRepository {
	return DefaultFileRepository{
		Cfg: cfg,
		files: &domain.SafeFileList{
			Files: make(map[string]domain.FileInfo),
		},
	}
}

// Exists checks whether a file identified by its path exists in the repository
func (fr DefaultFileRepository) Exists(filePath string) bool {
	fr.files.RLock()
	defer fr.files.RUnlock()
	_, ok := fr.files.Files[filePath]
	return ok
}

// Size returns the number of files stored in the repository
func (fr DefaultFileRepository) Size() int {
	fr.files.RLock()
	defer fr.files.RUnlock()
	return len(fr.files.Files)
}

// SizeOfType returns the number of files of the specified fileType
func (fr DefaultFileRepository) sizeOfType(fileType domain.FileType) (count int) {
	fr.files.RLock()
	defer fr.files.RUnlock()
	for _, f := range fr.files.Files {
		if f.FileType == fileType {
			count++
		}
	}
	return
}

// AudioSize returns the number of audio files (as identified by their file extension) stored in the repository
func (fr DefaultFileRepository) AudioSize() int {
	return fr.sizeOfType(domain.FileTypeAudio)
}

// StreamSize returns the number of stream files (as identified by their file extension) stored in the repository
func (fr DefaultFileRepository) StreamSize() int {
	return fr.sizeOfType(domain.FileTypeStream)
}

// GetByPath returns a file's information where the file is identified by its path. If no file matches, the methods returns nil
func (fr DefaultFileRepository) GetByPath(filePath string) *domain.FileInfo {
	var fi domain.FileInfo
	if !fr.Exists(filePath) {
		return nil
	}
	fr.files.RLock()
	defer fr.files.RUnlock()
	fi = fr.files.Files[filePath]
	return &fi
}

// GetByEventId returns a file's information where the file is identified by its event id (from calCMS). If no file matches, the methods returns nil
func (fr DefaultFileRepository) GetByEventId(eventId int) domain.FileList {
	var list domain.FileList
	if fr.Size() == 0 {
		return nil
	}
	fr.files.RLock()
	defer fr.files.RUnlock()
	for _, file := range fr.files.Files {
		if file.EventId == eventId {
			list = append(list, file)
		}
	}
	if len(list) > 0 {
		return list
	}
	return nil
}

// GetByEventIdAndDate returns file data matching a calCMS event id and folder date.
func (fr DefaultFileRepository) GetByEventIdAndDate(eventId int, folderDate time.Time) domain.FileList {
	var list domain.FileList
	if fr.Size() == 0 {
		return nil
	}
	normalizedDate := domain.NormalizeDate(folderDate)
	fr.files.RLock()
	defer fr.files.RUnlock()
	for _, file := range fr.files.Files {
		if file.EventId == eventId && file.FolderDate.Equal(normalizedDate) {
			list = append(list, file)
		}
	}
	if len(list) > 0 {
		return list
	}
	return nil
}

// GetByDate returns all file data from the repository for a specific folder date. Returns nil if repository is empty or no files match
func (fr DefaultFileRepository) GetByDate(folderDate time.Time) domain.FileList {
	var list domain.FileList
	if fr.Size() == 0 {
		return nil
	}
	fr.files.RLock()
	defer fr.files.RUnlock()
	for _, file := range fr.files.Files {
		if file.FolderDate.Equal(domain.NormalizeDate(folderDate)) {
			list = append(list, file)
		}
	}
	if len(list) > 0 {
		return list
	}
	return nil
}

// GetAll returns all file data from the repository. Returns nil if repository is empty
func (fr DefaultFileRepository) GetAll() domain.FileList {
	var list domain.FileList
	if fr.Size() == 0 {
		return nil
	}
	fr.files.RLock()
	defer fr.files.RUnlock()
	for _, file := range fr.files.Files {
		list = append(list, file)
	}
	return list
}

// GetByHour returns all files' information that fall into a given start hour. If no files match, the methods returns nil
func (fr DefaultFileRepository) GetByHour(hour string, includeLive bool) domain.FileList {
	return fr.GetByDateAndHour(helper.DateForFolder(fr.Cfg.Misc.TestCrawl, fr.Cfg.Misc.TestDate, 0), hour, includeLive)
}

// GetByDateAndHour returns all files for a folder date that fall into a given start hour.
func (fr DefaultFileRepository) GetByDateAndHour(folderDate time.Time, hour string, includeLive bool) domain.FileList {
	var list domain.FileList
	if fr.Size() == 0 {
		return nil
	}
	hi, err := strconv.Atoi(hour)
	if err != nil {
		return nil
	}
	normalizedDate := domain.NormalizeDate(folderDate)
	fr.files.RLock()
	defer fr.files.RUnlock()
	for _, file := range fr.files.Files {
		if (!file.StartTime.IsZero()) && (file.StartTime.Hour() == hi) && file.FolderDate.Equal(normalizedDate) {
			if (!file.EventIsLive) || (file.EventIsLive && includeLive) {
				list = append(list, file)
			}
		}
	}
	if len(list) > 0 {
		return list
	}
	return nil
}

// mergeFileList combines two file lists
func mergeFileList(fl1, fl2 domain.FileList) domain.FileList {
	var list domain.FileList
	if fl1 != nil {
		if len(fl1) > 0 {
			for _, f := range fl1 {
				list = append(list, f)
			}
		}
	}
	if fl2 != nil {
		if len(fl2) > 0 {
			for _, f := range fl2 {
				if !list.ContainsPath(f.Path) {
					list = append(list, f)
				}
			}
		}
	}
	return list
}

// GetByIdAndHour gets all elements form the list that either match an eventId or a particular hour
func (fr DefaultFileRepository) GetByIdAndHour(eventId int, hour string, includeLive bool) domain.FileList {
	return fr.GetByIdAndDateAndHour(eventId, helper.DateForFolder(fr.Cfg.Misc.TestCrawl, fr.Cfg.Misc.TestDate, 0), hour, includeLive)
}

// GetByIdAndDateAndHour gets all elements that either match an eventId or a particular hour on the same folder date.
func (fr DefaultFileRepository) GetByIdAndDateAndHour(eventId int, folderDate time.Time, hour string, includeLive bool) domain.FileList {
	var list domain.FileList
	files1 := fr.GetByEventIdAndDate(eventId, folderDate)
	files2 := fr.GetByDateAndHour(folderDate, hour, includeLive)
	list = mergeFileList(files1, files2)
	if len(list) > 0 {
		return list
	}
	return nil
}

// Store stores a new file information entry into the repository
func (fr DefaultFileRepository) Store(fi domain.FileInfo) error {
	if fi.Path == "" {
		return errors.New("cannot add item with empty path to list")
	}
	fr.files.Lock()
	defer fr.files.Unlock()
	fr.files.Files[fi.Path] = fi
	return nil
}

// Delete delete a file information entry from the repository, if it exists
func (fr DefaultFileRepository) Delete(filePath string) error {
	if !fr.Exists(filePath) {
		return fmt.Errorf("item with path %v does not exist", filePath)
	}
	fr.files.Lock()
	defer fr.files.Unlock()
	delete(fr.files.Files, filePath)
	return nil
}

// SaveToDisk writes the repository's contents to a specified file on disk
func (fr DefaultFileRepository) SaveToDisk(fileName string) error {
	logger.Info("Saving files data to disk...")
	fr.files.RLock()
	defer fr.files.RUnlock()
	b, err := json.Marshal(fr.files.Files)
	if err != nil {
		logger.Error("Error while converting file list to JSON", err)
		return err
	}
	if err := writeFileAtomic(fileName, b, 0644); err != nil {
		logger.Error("Error while writing files data to disk", err)
		return err
	}
	logger.Infof("Saved files data to disk (%v items)", len(fr.files.Files))
	return nil
}

func writeFileAtomic(fileName string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(fileName)
	base := filepath.Base(fileName)
	tmpFile, err := os.CreateTemp(dir, base+".*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmpFile.Name()
	defer os.Remove(tmpName)
	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return err
	}
	if err := tmpFile.Chmod(perm); err != nil {
		tmpFile.Close()
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, fileName); err != nil {
		if removeErr := os.Remove(fileName); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return removeErr
		}
		if renameErr := os.Rename(tmpName, fileName); renameErr != nil {
			return renameErr
		}
	}
	return nil
}

// LoadFromDisk loads file information stored on disk into memory
func (fr DefaultFileRepository) LoadFromDisk(fileName string) error {
	logger.Info("Reading files data from disk...")
	fileDta := make(map[string]domain.FileInfo)
	b, err := os.ReadFile(fileName)
	if err != nil {
		logger.Error("Error while reading files data from disk", err)
		return err
	}
	if err := json.Unmarshal(b, &fileDta); err != nil {
		logger.Error("Error while converting files data to json", err)
		return err
	}
	fr.files.Lock()
	defer fr.files.Unlock()
	fr.files.Files = fileDta
	logger.Infof("Read files data from disk (%v items)", len(fr.files.Files))
	return nil
}

// DeleteAllData removes all entries from the repository
func (fr DefaultFileRepository) DeleteAllData() {
	fr.files.Lock()
	defer fr.files.Unlock()
	fr.files.Files = make(map[string]domain.FileInfo)
}

// NewFiles returns true, if there are file entries in the repository for which additional information hasn't been extracted
func (fr DefaultFileRepository) NewFiles() (newFiles bool) {
	if fr.Size() > 0 {
		allFiles := fr.GetAll()
		for _, file := range allFiles {
			if !file.InfoExtracted {
				newFiles = true
				break
			}
		}
	}
	return
}
