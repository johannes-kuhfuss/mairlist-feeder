// package service implements the services and their business logic that provide the main part of the program
package service

import (
	"errors"
	"sync"
	"time"

	"github.com/johannes-kuhfuss/mairlist-feeder/appstate"
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
	Cfg   *config.AppConfig
	State *appstate.AppState
	Repo  *repositories.DefaultFileRepository
	Now   func() time.Time
	mu    *sync.Mutex
}

// NewCleanService creates a new cleaning service and injects its dependencies
func NewCleanService(cfg *config.AppConfig, repo *repositories.DefaultFileRepository) DefaultCleanService {
	return NewCleanServiceWithState(cfg, appstate.New(), repo)
}

func NewCleanServiceWithState(cfg *config.AppConfig, state *appstate.AppState, repo *repositories.DefaultFileRepository) DefaultCleanService {
	return DefaultCleanService{
		Cfg:   cfg,
		State: state,
		Repo:  repo,
		Now:   time.Now,
		mu:    &sync.Mutex{},
	}
}

// isYesterdayOrOlder is a helper function which checks each entry and determines whether this entry can be purged from the list
func isYesterdayOrOlder(folderDate time.Time) (bool, error) {
	return isYesterdayOrOlderAt(folderDate, time.Now())
}

func isYesterdayOrOlderAt(folderDate time.Time, now time.Time) (bool, error) {
	if folderDate.IsZero() {
		return false, errors.New("folder date is empty")
	}
	fileDate := time.Date(folderDate.Year(), folderDate.Month(), folderDate.Day(), 0, 0, 0, 0, time.Local)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	diff := today.Sub(fileDate).Hours() / 24
	return diff >= 1, nil
}

// Clean orchestrates the clean-up of the file list kept in memory
func (s DefaultCleanService) Clean() (err error) {
	runStart := s.Now()
	defer func() {
		recordRunMetrics(s.State, "clean", runStart, err)
	}()
	logger.Info("Starting file list clean-up...")
	start := s.Now().UTC()
	var filesCleaned int
	filesCleaned, err = s.CleanRun()
	if err != nil {
		logger.Error("Error while cleaning repository", err)
	}
	s.State.Runtime.Mu.Lock()
	s.State.Runtime.FilesCleaned = filesCleaned
	s.State.Runtime.Mu.Unlock()
	end := s.Now().UTC()
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
	s.State.Runtime.Mu.Lock()
	s.State.Runtime.CleanRunning = true
	s.State.Runtime.LastCleanDate = s.Now()
	s.State.Runtime.Mu.Unlock()
	defer func() {
		s.State.Runtime.Mu.Lock()
		s.State.Runtime.CleanRunning = false
		s.State.Runtime.Mu.Unlock()
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
		fromYesterday, err := isYesterdayOrOlderAt(file.FolderDate, s.Now())
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
