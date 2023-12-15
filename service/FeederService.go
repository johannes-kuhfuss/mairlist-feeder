package service

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"

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
	rawFileList domain.FileList
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
	rootFolder := s.Cfg.MAirList.RootFolder
	err := crawlFolder(rootFolder, s.Cfg.MAirList.Extensions)
	if err != nil {
		logger.Error(fmt.Sprintf("Error crawling folder %v: ", rootFolder), err)
	}
	logger.Info(fmt.Sprintf("Number of entries in file list: %v", len(rawFileList)))
	exp := regexp.MustCompile("(0[0-9]|1[0-9]|2[0-3])-(0[0-9]|[1-5][0-9])")
	for i, file := range rawFileList {
		//logger.Info(fmt.Sprintf("Index: %v, File: %v - Modification Time: %v - Size: %v", i, file.FilePath, file.FileInfo.ModTime(), file.FileInfo.Size()))
		folder := filepath.Dir(file.FilePath)
		if exp.MatchString(folder) {
			logger.Info(fmt.Sprintf("Index: %v, File: %v", i, file.FilePath))
		}
	}
}

func crawlFolder(rootFolder string, extensions []string) error {
	var fi domain.FileInfo
	err := filepath.Walk(getTodayFolder(rootFolder),
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if misc.SliceContainsString(extensions, filepath.Ext(path)) {
				fi.FilePath = path
				fi.FileInfo = info
				rawFileList = append(rawFileList, fi)
			}
			return nil
		})
	if err != nil {
		return err
	} else {
		return nil
	}
}

func getTodayFolder(rootFolder string) string {
	/*
		year := fmt.Sprintf("%d", time.Now().Year())
		month := fmt.Sprintf("%02d", time.Now().Month())
		day := fmt.Sprintf("%02d", time.Now().Day())

		return path.Join(rootFolder, year, month, day)
	*/
	return path.Join(rootFolder, "2023", "12")
}
