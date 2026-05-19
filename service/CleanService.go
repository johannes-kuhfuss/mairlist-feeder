// package service implements the services and their business logic that provide the main part of the program
package service

import (
	"errors"
	"sync"
	"time"

	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/domain"
	"github.com/johannes-kuhfuss/mairlist-feeder/repositories"
	"github.com/johannes-kuhfuss/services_utils/logger"
)

type Cleaner interface {
	Clean() error
}

// The clean service handles the cyclical clean-up of the file list kept in memory
type DefaultCleanService struct {
	Cfg  *config.AppConfig
	Repo *repositories.DefaultFileRepository
	mu   *sync.Mutex
}

// NewCleanService creates a new cleaning service and injects its dependencies
func NewCleanService(cfg *config.AppConfig, repo *repositories.DefaultFileRepository) DefaultCleanService {
	return DefaultCleanService{
		Cfg:  cfg,
		Repo: repo,
		mu:   &sync.Mutex{},
	}
}

// isYesterdayOrOlder is a helper function which checks each entry and determines whether this entry can be purged from the list
func isYesterdayOrOlder(folderDate time.Time) (bool, error) {
	if folderDate.IsZero() {
		return false, errors.New("folder date is empty")
	}
	fileDate := time.Date(folderDate.Year(), folderDate.Month(), folderDate.Day(), 0, 0, 0, 0, time.Local)
	today := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Local)
	diff := today.Sub(fileDate).Hours() / 24
	return diff >= 1, nil
}

// Clean orchestrates the clean-up of the file list kept in memory
func (s DefaultCleanService) Clean() (err error) {
	defer func() {
		recordRunResult(s.Cfg, "clean", err)
	}()
	logger.Info("Starting file list clean-up...")
	start := time.Now().UTC()
	var filesCleaned int
	filesCleaned, err = s.CleanRun()
	if err != nil {
		logger.Error("Error while cleaning repository", err)
	}
	s.Cfg.RunTime.Mu.Lock()
	s.Cfg.RunTime.FilesCleaned = filesCleaned
	s.Cfg.RunTime.Mu.Unlock()
	end := time.Now().UTC()
	dur := end.Sub(start)
	logger.Infof("File list cleaned-up. Removed %v entries. (%v)", filesCleaned, dur.String())
	return err
}

// CleanRun performs the clean-up of expired file list entries
func (s DefaultCleanService) CleanRun() (filesCleaned int, e error) {
	var (
		errorCounter int
	)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Cfg.RunTime.Mu.Lock()
	s.Cfg.RunTime.CleanRunning = true
	s.Cfg.RunTime.LastCleanDate = time.Now()
	s.Cfg.RunTime.Mu.Unlock()
	defer func() {
		s.Cfg.RunTime.Mu.Lock()
		s.Cfg.RunTime.CleanRunning = false
		s.Cfg.RunTime.Mu.Unlock()
	}()
	if files := s.Repo.GetAll(); files != nil {
		filesCleaned, errorCounter = s.checkAndClean(files)
	}
	if errorCounter == 0 {
		return filesCleaned, nil
	} else {
		return filesCleaned, errors.New("error cleaning")
	}
}

// checkAndClean checks the folder date of each file and, if older than today, removes the file from the in-memory store
func (s DefaultCleanService) checkAndClean(files *domain.FileList) (fileCount int, errorCount int) {
	for _, file := range *files {
		fromYesterday, err := isYesterdayOrOlder(file.FolderDate)
		if err != nil {
			errorCount++
			logger.Error("Error converting date", err)
		}
		if fromYesterday {
			logger.Infof("Removing entry for expired file %v", file.Path)
			if err := s.Repo.Delete(file.Path); err != nil {
				errorCount++
				logger.Error("Could not remove entry", err)
			} else {
				fileCount++
			}
		}
	}
	return
}
