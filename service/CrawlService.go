package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/domain"
	"github.com/johannes-kuhfuss/mairlist-feeder/repositories"
	"github.com/johannes-kuhfuss/services_utils/logger"
	"github.com/johannes-kuhfuss/services_utils/misc"
)

type CrawlService interface {
	Crawl()
}

type DefaultCrawlService struct {
	Cfg  *config.AppConfig
	Repo *repositories.DefaultFileRepository
}

func NewCrawlService(cfg *config.AppConfig, repo *repositories.DefaultFileRepository) DefaultCrawlService {
	return DefaultCrawlService{
		Cfg:  cfg,
		Repo: repo,
	}
}

func (s DefaultCrawlService) Crawl() {
	if s.Cfg.Crawl.RootFolder == "" {
		logger.Warn("No root folder given. Not running")
		s.Cfg.RunTime.RunFeeder = false
	} else {
		logger.Info(fmt.Sprintf("Root folder defined as %v. Starting to crawl.", s.Cfg.Crawl.RootFolder))
		s.Cfg.RunTime.RunFeeder = true
	}
	s.CrawlRun()
}

func (s DefaultCrawlService) CrawlRun() {
	rootFolder := s.Cfg.Crawl.RootFolder
	s.Cfg.RunTime.CrawlRunNumber++
	logger.Info(fmt.Sprintf("Starting crawl run #%v...", s.Cfg.RunTime.CrawlRunNumber))
	err := s.crawlFolder(rootFolder, s.Cfg.Crawl.Extensions)
	if err != nil {
		logger.Error(fmt.Sprintf("Error crawling folder %v: ", rootFolder), err)
	}
	logger.Info(fmt.Sprintf("Finished crawl run #%v.", s.Cfg.RunTime.CrawlRunNumber))
	numEntries := s.Repo.Size()
	logger.Info(fmt.Sprintf("Number of entries in file list: %v", numEntries))
	if numEntries > 0 {
		logger.Info("Starting to extract file data...")
		s.extractFileInfo()
		logger.Info("Finished extracting file data.")
	} else {
		logger.Info("No files in file list. No extraction needed.")
	}

}

func (s DefaultCrawlService) crawlFolder(rootFolder string, extensions []string) error {
	var fi domain.FileInfo
	today := getTodayFolder()
	err := filepath.Walk(path.Join(rootFolder, today),
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if misc.SliceContainsString(extensions, filepath.Ext(path)) {
				if s.Repo.Exists(path) {
					oldFile := s.Repo.Get(path)
					if oldFile.FileInfo.ModTime() == info.ModTime() {
						logger.Info(fmt.Sprintf("File %v already exists and is unmodified. Not adding", path))
						return nil
					}
				}
				fi.FileInfo = info
				fi.Path = path
				fi.FromCalCMS = false
				fi.ScanTime = time.Now()
				fi.FolderDate = strings.Replace(today, "/", "-", -1)
				fi.InfoExtracted = false
				s.Repo.Store(fi)
				if s.Cfg.Misc.Test {
					logger.Info(fmt.Sprintf("File %v added", path))
				}
			}
			return nil
		})
	if err != nil {
		return err
	} else {
		return nil
	}
}

func (s DefaultCrawlService) extractFileInfo() error {
	var startTimeDisplay string
	folderExp := regexp.MustCompile(`[\\/]+(0[0-9]|1[0-9]|2[0-3])-(0[0-9]|[1-5][0-9])`)
	file1Exp := regexp.MustCompile(`([01][0-9]|2[0-3])[0-5][0]\s?-\s?([01][0-9]|2[0-3])[0-5][0][_ -]`)
	file2Exp := regexp.MustCompile(`([01][0-9]|2[0-3])[0-5][0][_ -]`)
	file3Exp := regexp.MustCompile(`([01][0-9]|2[0-3])[_-]([01][0-9]|2[0-3])[_ -]?Uhr`)
	files := s.Repo.GetAll()
	if files != nil {
		for _, file := range *files {
			if !file.InfoExtracted {
				var timeData string
				var newInfo domain.FileInfo = file

				len, err := analyzeLength(file.Path, s.Cfg.Crawl.FfProbeTimeOut, s.Cfg.Crawl.FfprobePath)
				if err != nil {
					logger.Error("Could not analyze file length: ", err)
				}
				newInfo.Duration = len
				folderName := filepath.Dir(file.Path)
				fileName := filepath.Base(file.Path)
				switch {
				// Case 1: file has been uploaded via calCMS, start time is coded in folder name: "\HH-MM" or "/HH-MM"
				case folderExp.MatchString(folderName):
					{
						timeData = folderExp.FindString(folderName)
						newInfo.FromCalCMS = true
						newInfo.StartTime = timeData[1:3] + ":" + timeData[4:6]
						newInfo.RuleMatched = "calCMS Folder Rule"
					}
				// Case 2: file has been uploaded manually, time slot is coded in file name in the form "HHMM-HHMM_"
				case file1Exp.MatchString(fileName):
					{
						timeData = file1Exp.FindString(fileName)
						timeData = strings.Replace(timeData, " ", "", -1)
						newInfo.StartTime = timeData[0:2] + ":" + timeData[2:4]
						newInfo.EndTime = timeData[5:7] + ":" + timeData[7:9]
						newInfo.RuleMatched = "Manual, File Name HHMM-HHMM"
					}
				// Case 3: file has been uploaded manually, start time is coded in file name in the form "HHMM_"
				case file2Exp.MatchString(fileName):
					{
						timeData = file2Exp.FindString(fileName)
						newInfo.StartTime = timeData[0:2] + ":" + timeData[2:4]
						newInfo.RuleMatched = "Manual, File Name HHMM"
					}
				// Case 4: file has been uploaded manually, start time is coded in file name in the form "HH-HH_Uhr"
				case file3Exp.MatchString(fileName):
					{
						timeData = file3Exp.FindString(fileName)
						timeData = strings.Replace(timeData, " ", "", -1)
						newInfo.StartTime = timeData[0:2] + ":00"
						newInfo.EndTime = timeData[3:5] + ":00"
						newInfo.RuleMatched = "Manual, File Name HH-HH Uhr"
					}
				default:
					{
						newInfo.RuleMatched = "None"
					}
				}
				newInfo.InfoExtracted = true
				err = s.Repo.Store(newInfo)
				if err != nil {
					logger.Error("Error while storing data: ", err)
				}
				if newInfo.StartTime == "" {
					startTimeDisplay = "N/A"
				} else {
					startTimeDisplay = newInfo.StartTime
				}
				logger.Info(fmt.Sprintf("Time Slot: % v, File: %v - Length (sec): %v", startTimeDisplay, file.Path, len))
			}
		}
	}
	return nil
}

func analyzeLength(path string, timeout int, ffprobe string) (len float64, err error) {
	ctx := context.Background()
	timeoutDuration := time.Duration(timeout) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeoutDuration)
	// Syntax: ffprobe -show_format -print_format json -loglevel quiet <input_file>
	cmd := exec.CommandContext(ctx, ffprobe, "-show_format", "-print_format", "json", "-loglevel", "quiet", path)
	outJson, err := cmd.CombinedOutput()
	if err != nil {
		cancel()
		logger.Error("Could not execute ffprobe: ", err)
		return 0, err
	}
	cancel()
	durationSec, err := parseDuration(outJson)
	if err != nil {
		logger.Error("Could not parse duration: ", err)
		return 0, err
	}
	return durationSec, nil
}

func parseDuration(ffprobedata []byte) (durationSec float64, err error) {
	var result domain.FfprobeResult
	err = json.Unmarshal(ffprobedata, &result)
	if err != nil {
		return 0, err
	}
	durFloat, err := strconv.ParseFloat(result.Format.Duration, 64)
	if err != nil {
		return 0, err
	}
	return durFloat, nil
}
