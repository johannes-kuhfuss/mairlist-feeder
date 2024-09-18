package service

import (
	"fmt"
	"time"

	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/repositories"
	"github.com/johannes-kuhfuss/services_utils/logger"
)

type CleanService interface {
	Clean()
}

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

func isYesterdayOrOlder(folderDate string) bool {
	const dateLayout = "2006-01-02"
	fileDate, err := time.Parse(dateLayout, folderDate)
	if err != nil {
		logger.Error("Could not convert date: ", err)
		return false
	}
	today := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC)
	diff := today.Sub(fileDate).Hours() / 24
	return diff >= 1
}

func (s DefaultCleanService) Clean() {
	var (
		filesCleaned int = 0
	)
	if s.Cfg.RunTime.CleanRunning {
		logger.Warn("Clean-up already running. Not starting another one.")
	} else {
		s.Cfg.RunTime.CleanRunning = true
		logger.Info("Starting file list clean-up...")
		files := s.Repo.GetAll()
		if files != nil {
			for _, file := range *files {
				if isYesterdayOrOlder(file.FolderDate) {
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
		logger.Info(fmt.Sprintf("File list clean-up done. Cleaned %v entries.", filesCleaned))
		s.Cfg.RunTime.CleanRunning = false
	}
}
