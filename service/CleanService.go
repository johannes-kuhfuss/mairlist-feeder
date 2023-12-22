package service

import (
	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/repositories"
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

}
