package service

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/domain"
	"github.com/johannes-kuhfuss/services_utils/logger"
	"github.com/johannes-kuhfuss/services_utils/misc"
)

type FeederService interface {
	Feed()
}

type DefaultFeederService struct {
	Cfg *config.AppConfig
}

var (
	fileList domain.FileList
)

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
		logger.Info(fmt.Sprintf("Starting to crawl folder %v", s.Cfg.MAirList.RootFolder))
		s.Cfg.RunTime.RunFeeder = true
	}
	/*
		for s.Cfg.RunTime.RunFeeder {
			FeedRun(s)
			time.Sleep(time.Duration(5 * time.Second))
		}
	*/
	FeedRun(s)
}

func FeedRun(s DefaultFeederService) {
	folder := s.Cfg.MAirList.RootFolder
	err := crawlFolder(folder, s.Cfg.MAirList.Extensions)
	if err != nil {
		logger.Error(fmt.Sprintf("Error crawling folder %v: ", folder), err)
	}
	for i, file := range fileList {
		logger.Info(fmt.Sprintf("Index: %v, File: %v - Modification Time: %v - Size: %v", i, file.FilePath, file.FileInfo.ModTime(), file.FileInfo.Size()))
	}
}

func crawlFolder(folder string, extensions []string) error {
	var fi domain.FileInfo
	err := filepath.Walk(folder,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if misc.SliceContainsString(extensions, filepath.Ext(path)) {
				fi.FilePath = path
				fi.FileInfo = info
				fileList = append(fileList, fi)
			}
			return nil
		})
	if err != nil {
		return err
	} else {
		return nil
	}
}
