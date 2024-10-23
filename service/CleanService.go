// package service implements the services and their business logic that provide the main part of the program
package service

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/repositories"
	"github.com/johannes-kuhfuss/services_utils/logger"
)

type CleanService interface {
	Clean()
}

var (
	clmu sync.Mutex
)

// The clean service handles the cyclical clean-up of the file list kept in memory
type DefaultCleanService struct {
	Cfg  *config.AppConfig
	Repo *repositories.DefaultFileRepository
}

// NewCleanService creates a new cleaning service and injects its dependencies
func NewCleanService(cfg *config.AppConfig, repo *repositories.DefaultFileRepository) DefaultCleanService {
	return DefaultCleanService{
		Cfg:  cfg,
		Repo: repo,
	}
}

// isYesterdayOrOlder is a helper function which checks each entry and determines whether this entry can be purged from the list
func isYesterdayOrOlder(folderDate string) (bool, error) {
	const dateLayout = "2006-01-02"
	fileDate, err := time.Parse(dateLayout, folderDate)
	if err != nil {
		logger.Error("Could not convert date: ", err)
		return false, err
	}
	today := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC)
	diff := today.Sub(fileDate).Hours() / 24
	return diff >= 1, nil
}

// Clean orchestrates the clean-up of the file list kept in memory
func (s DefaultCleanService) Clean() {
	logger.Info("Starting file list clean-up...")
	filesCleaned, err := s.CleanRun()
	if err != nil {
		logger.Error("Error while clean repository", err)
	}
	s.Cfg.RunTime.FilesCleaned = filesCleaned
	logger.Info(fmt.Sprintf("File list clean-up done. Cleaned %v entries.", filesCleaned))
}

// CleanRun performs the clean-up of expired file list entries
func (s DefaultCleanService) CleanRun() (filesCleaned int, e error) {
	var (
		errorCounter int
	)
	clmu.Lock()
	defer clmu.Unlock()
	s.Cfg.RunTime.CleanRunning = true
	s.Cfg.RunTime.LastCleanDate = time.Now()
	if files := s.Repo.GetAll(); files != nil {
		for _, file := range *files {
			fromYesterday, err := isYesterdayOrOlder(file.FolderDate)
			if err != nil {
				errorCounter++
				logger.Error("Error converting date", err)
			}
			if fromYesterday {
				logger.Info(fmt.Sprintf("Removing entry for expired file %v", file.Path))
				if err := s.Repo.Delete(file.Path); err != nil {
					errorCounter++
					logger.Error("Could not remove entry: ", err)
				} else {
					filesCleaned++
				}
			}
		}
	}
	s.Cfg.RunTime.CleanRunning = false
	if errorCounter == 0 {
		return filesCleaned, nil
	} else {
		return filesCleaned, errors.New("error cleaning")
	}
}
