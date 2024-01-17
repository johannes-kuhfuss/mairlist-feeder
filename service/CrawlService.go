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
	"github.com/johannes-kuhfuss/mairlist-feeder/helper"
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
		s.Cfg.RunTime.RunFeeder = true
	}
	s.CrawlRun()
}

func (s DefaultCrawlService) CrawlRun() {
	rootFolder := s.Cfg.Crawl.RootFolder
	s.Cfg.RunTime.CrawlRunNumber++
	logger.Info(fmt.Sprintf("Root folder: %v. Starting crawl #%v.", s.Cfg.Crawl.RootFolder, s.Cfg.RunTime.CrawlRunNumber))
	fileCount, err := s.crawlFolder(rootFolder, s.Cfg.Crawl.Extensions)
	if err != nil {
		logger.Error(fmt.Sprintf("Error crawling folder %v: ", rootFolder), err)
	}
	logger.Info(fmt.Sprintf("Finished crawl run #%v. Added %v new files.", s.Cfg.RunTime.CrawlRunNumber, fileCount))
	if s.Repo.NewFiles() {
		logger.Info("Starting to extract file data...")
		fileCount, _ := s.extractFileInfo()
		logger.Info(fmt.Sprintf("Finished extracting file data for %v files.", fileCount))
	} else {
		logger.Info("No (new) files in file list. No extraction needed.")
	}

}

func (s DefaultCrawlService) crawlFolder(rootFolder string, extensions []string) (int, error) {
	var (
		fi        domain.FileInfo
		fileCount int = 0
	)
	today := helper.GetTodayFolder(s.Cfg.Misc.Test, s.Cfg.Misc.TestDate)
	err := filepath.Walk(path.Join(rootFolder, today),
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if misc.SliceContainsString(extensions, filepath.Ext(path)) {
				if s.Repo.Exists(path) {
					oldFile := s.Repo.Get(path)
					if oldFile.ModTime == info.ModTime() {
						logger.Info(fmt.Sprintf("File %v already exists and is unmodified. Not adding", path))
						return nil
					}
				}
				fi.ModTime = info.ModTime()
				fi.Path = path
				fi.FromCalCMS = false
				fi.ScanTime = time.Now()
				fi.FolderDate = strings.Replace(today, "/", "-", -1)
				fi.InfoExtracted = false
				fileCount++
				s.Repo.Store(fi)
				if s.Cfg.Misc.Test {
					logger.Info(fmt.Sprintf("File %v added", path))
				}
			}
			return nil
		})
	if err != nil {
		return fileCount, err
	} else {
		return fileCount, err
	}
}

func (s DefaultCrawlService) extractFileInfo() (int, error) {
	var (
		startTimeDisplay string
		extractCount     int = 0
	)
	// /HH-MM
	folder1Exp := regexp.MustCompile(`[\\/]+([01][0-9]|2[0-3])-(0[0-9]|[1-5][0-9])`)
	// /HHMM-HHMM
	folder2Exp := regexp.MustCompile(`[\\/]+([01][0-9]|2[0-3])(0[0-9]|[1-5][0-9])\s?-\s?([01][0-9]|2[0-3])(0[0-9]|[1-5][0-9])`)
	// /HH bis HH
	folder3Exp := regexp.MustCompile(`[\\/]+([01][0-9]|2[0-3])\s?bis\s?([01][0-9]|2[0-3])`)
	// HHMM-HHMM
	file1Exp := regexp.MustCompile(`^([01][0-9]|2[0-3])(0[0-9]|[1-5][0-9])\s?-\s?([01][0-9]|2[0-3])(0[0-9]|[1-5][0-9])[_ -]`)
	// HH-HH_Uhr
	file2Exp := regexp.MustCompile(`([01][0-9]|2[0-3])[_-]([01][0-9]|2[0-3])[_ -]?Uhr`)
	// HHMM_
	file3Exp := regexp.MustCompile(`^([01][0-9]|2[0-3])(0[0-9]|[1-5][0-9])[_ -][a-zA-Z]+`)
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
				// Condition: only start time is encoded in folder name: "/HH-MM" (calCMS)
				case folder1Exp.MatchString(folderName):
					{
						timeData = folder1Exp.FindString(folderName)
						newInfo.FromCalCMS = true
						newInfo.StartTime = timeData[1:3] + ":" + timeData[4:6]
						newInfo.RuleMatched = "HH-MM folder name rule (calcms)"
					}
				// Condition: start time and end time is encoded in folder name: "/HHMM-HHMM"
				case folder2Exp.MatchString(folderName):
					{
						timeData = folder2Exp.FindString(folderName)
						timeData = strings.Replace(timeData, " ", "", -1)
						newInfo.FromCalCMS = true
						newInfo.StartTime = timeData[1:3] + ":" + timeData[3:5]
						newInfo.EndTime = timeData[6:8] + ":" + timeData[8:10]
						newInfo.RuleMatched = "HHMM-HHMM folder name rule"
					}
				// Condition: start time (hour) and end time (hour) is encoded in folder name: "HH bis HH"
				case folder3Exp.MatchString(folderName):
					{
						timeData = folder3Exp.FindString(folderName)
						timeData = strings.Replace(timeData, " ", "", -1)
						newInfo.FromCalCMS = true
						newInfo.StartTime = timeData[1:3] + ":00"
						newInfo.EndTime = timeData[6:8] + ":00"
						newInfo.RuleMatched = "HH bis HH folder name rule"
					}
				// Condition: start time and end time is encoded in file name in the form "HHMM-HHMM_"
				case file1Exp.MatchString(fileName):
					{
						timeData = file1Exp.FindString(fileName)
						timeData = strings.Replace(timeData, " ", "", -1)
						newInfo.StartTime = timeData[0:2] + ":" + timeData[2:4]
						newInfo.EndTime = timeData[5:7] + ":" + timeData[7:9]
						newInfo.RuleMatched = "HHMM-HHMM file name rule"
					}
				// Condition: start time (hour) and end time (hour) is encoded in file name in the form "HH-HH_Uhr"
				case file2Exp.MatchString(fileName):
					{
						timeData = file2Exp.FindString(fileName)
						timeData = strings.Replace(timeData, " ", "", -1)
						newInfo.StartTime = timeData[0:2] + ":00"
						newInfo.EndTime = timeData[3:5] + ":00"
						newInfo.RuleMatched = "HH-HH Uhr file name rule"
					}
				// Condition: only start time is encoded in the file name in the form of "HHMM_" (beware of date matching!)
				case file3Exp.MatchString(fileName):
					{
						timeData = file3Exp.FindString(fileName)
						newInfo.StartTime = timeData[0:2] + ":" + timeData[2:4]
						newInfo.RuleMatched = "HHMM_ file name rule"
					}
				default:
					{
						newInfo.RuleMatched = "None"
					}
				}
				newInfo.InfoExtracted = true
				extractCount++
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
	return extractCount, nil
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
