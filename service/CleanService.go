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

func (s DefaultCleanService) Clean() {
	var (
		filesCleaned int = 0
	)
	if s.Cfg.RunTime.CleanRunning {
		logger.Warn("Clean-up already running. Not starting another one.")
	} else {
		s.Cfg.RunTime.CleanRunning = true
		logger.Info("Starting file list clean-up...")
		const dateLayout = "2006-01-02"
		cleanDate := time.Now().AddDate(0, 0, -1)
		files := s.Repo.GetAll()
		if files != nil {
			for _, file := range *files {
				fileDate, err := time.Parse(dateLayout, file.FolderDate)
				if err != nil {
					logger.Error("Could not convert date: ", err)
				}
				if cleanDate.Sub(fileDate) >= time.Duration(24*time.Hour) {
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
