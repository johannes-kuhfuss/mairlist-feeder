// package service implements the services and their business logic that provide the main part of the program
package service

import (
	"context"
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
	CleanContext(context.Context) error
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
	location := now.Location()
	fileDate := time.Date(folderDate.Year(), folderDate.Month(), folderDate.Day(), 0, 0, 0, 0, location)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
	return fileDate.Before(today), nil
}

// Clean orchestrates the clean-up of the file list kept in memory
func (s DefaultCleanService) Clean() (err error) {
	return s.CleanContext(context.Background())
}

func (s DefaultCleanService) CleanContext(ctx context.Context) (err error) {
	runStart := s.Now()
	defer func() {
		recordRunMetrics(s.State, "clean", runStart, err)
	}()
	logger.Info("Starting file list clean-up...")
	start := s.Now().UTC()
	var filesCleaned int
	filesCleaned, err = s.CleanRunContext(ctx)
	if err != nil {
		logger.Error("Error while cleaning repository", err)
	}
	s.State.Runtime.Update(func(runtime *appstate.RuntimeState) { runtime.FilesCleaned = filesCleaned })
	end := s.Now().UTC()
	dur := end.Sub(start)
	logger.Infof("File list cleaned-up. Removed %v entries. (%v)", filesCleaned, dur.String())
	return err
}

// CleanRun performs the clean-up of expired file list entries
func (s DefaultCleanService) CleanRun() (filesCleaned int, e error) {
	return s.CleanRunContext(context.Background())
}

func (s DefaultCleanService) CleanRunContext(ctx context.Context) (filesCleaned int, e error) {
	var (
		errorCounter int
	)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.State.Runtime.Update(func(runtime *appstate.RuntimeState) {
		runtime.CleanRunning = true
		runtime.LastCleanDate = s.Now()
	})
	defer func() {
		s.State.Runtime.Update(func(runtime *appstate.RuntimeState) { runtime.CleanRunning = false })
	}()
	if files := s.Repo.GetAll(); files != nil {
		filesCleaned, errorCounter = s.checkAndClean(ctx, files)
	}
	if errorCounter == 0 {
		return filesCleaned, nil
	} else {
		return filesCleaned, errors.New("error cleaning")
	}
}

// checkAndClean checks the folder date of each file and, if older than today, removes the file from the in-memory store
func (s DefaultCleanService) checkAndClean(ctx context.Context, files domain.FileList) (fileCount int, errorCount int) {
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return fileCount, errorCount + 1
		}
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
