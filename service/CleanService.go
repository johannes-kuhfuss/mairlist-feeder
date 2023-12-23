package service

import (
	"fmt"

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
	logger.Info("Starting file list clean-up...")
	files := s.Repo.GetAll()
	if files != nil {
		for _, file := range *files {
			logger.Info(fmt.Sprintf("File: %v", file.Path))

		}
	}
	logger.Info("File list clean-up done")
}
