package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/helper"
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
		cleanCounter int = 0
	)
	logger.Info("Starting file list clean-up...")
	const dateLayout = "2006-01-02"
	today, err := time.Parse(dateLayout, strings.Replace(helper.GetTodayFolder(false, ""), "/", "-", -1))
	if err != nil {
		logger.Error("Could not convert date: ", err)
	}
	files := s.Repo.GetAll()
	if files != nil {
		for _, file := range *files {
			fileDate, err := time.Parse(dateLayout, file.FolderDate)
			if err != nil {
				logger.Error("Could not convert date: ", err)
			}
			if today.Sub(fileDate) >= time.Duration(24*time.Hour) {
				logger.Info(fmt.Sprintf("Removing entry for expired file %v", file.Path))
				err := s.Repo.Delete(file.Path)
				if err != nil {
					logger.Error("Could not remove entry: ", err)
				} else {
					cleanCounter++
				}
			}
		}
	}
	logger.Info(fmt.Sprintf("File list clean-up done. Cleaned %v entries.", cleanCounter))
}
