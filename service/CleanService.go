package service

import (
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

type DefaultCleanService struct {
	Cfg  *config.AppConfig
	Repo *repositories.DefaultFileRepository
}

func NewCleanService(cfg *config.AppConfig, repo *repositories.DefaultFileRepository) DefaultCleanService {
	return DefaultCleanService{
		Cfg:  cfg,
		Repo: repo,
	}
}

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

func (s DefaultCleanService) Clean() {
	var (
		filesCleaned int = 0
	)
	clmu.Lock()
	defer clmu.Unlock()
	s.Cfg.RunTime.CleanRunning = true
	s.Cfg.RunTime.LastCleanDate = time.Now()
	logger.Info("Starting file list clean-up...")
	files := s.Repo.GetAll()
	if files != nil {
		for _, file := range *files {
			yesterday, err := isYesterdayOrOlder(file.FolderDate)
			if err != nil {
				logger.Error("Error converting date", err)
			}
			if yesterday {
				logger.Info(fmt.Sprintf("Removing entry for expired file %v", file.Path))
				err := s.Repo.Delete(file.Path)
				if err != nil {
					logger.Error("Could not remove entry: ", err)
				} else {
					filesCleaned++
				}
			}
		}
	}
	s.Cfg.RunTime.CleanRunning = false
	logger.Info(fmt.Sprintf("File list clean-up done. Cleaned %v entries.", filesCleaned))
}
