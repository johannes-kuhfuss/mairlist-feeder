package service

import (
	"fmt"
	"time"

	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/services_utils/logger"
)

type FeederService interface {
	Feed()
}

type DefaultFeederService struct {
	Cfg *config.AppConfig
}

var ()

func NewFeederService(cfg *config.AppConfig) DefaultFeederService {
	return DefaultFeederService{
		Cfg: cfg,
	}
}

func (s DefaultFeederService) Feed() {
	if s.Cfg.MAirList.RootFolder == "" {
		logger.Warn("No root folder given. Not running")
		s.Cfg.RunTime.RunFeeder = false
	} else {
		logger.Info(fmt.Sprintf("Starting to crawl root folder %v", s.Cfg.MAirList.RootFolder))
		s.Cfg.RunTime.RunFeeder = true
	}

	for s.Cfg.RunTime.RunFeeder {
		FeedRun(s)
		time.Sleep(time.Duration(5 * time.Second))
	}
}

func FeedRun(s DefaultFeederService) {
	logger.Info("Running feed")
}
