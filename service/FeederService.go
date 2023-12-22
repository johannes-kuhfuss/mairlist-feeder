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
	"time"

	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/domain"
	"github.com/johannes-kuhfuss/mairlist-feeder/repositories"
	"github.com/johannes-kuhfuss/services_utils/logger"
	"github.com/johannes-kuhfuss/services_utils/misc"
)

type FeederService interface {
	Feed()
}

type DefaultFeederService struct {
	Cfg  *config.AppConfig
	Repo *repositories.DefaultFileRepository
}

func NewFeederService(cfg *config.AppConfig, repo *repositories.DefaultFileRepository) DefaultFeederService {
	return DefaultFeederService{
		Cfg:  cfg,
		Repo: repo,
	}
}

func (s DefaultFeederService) Feed() {
	if s.Cfg.Crawl.RootFolder == "" {
		logger.Warn("No root folder given. Not running")
		s.Cfg.RunTime.RunFeeder = false
	} else {
		logger.Info(fmt.Sprintf("Root folder defined as %v. Starting to crawl.", s.Cfg.Crawl.RootFolder))
		s.Cfg.RunTime.RunFeeder = true
	}
	for s.Cfg.RunTime.RunFeeder {
		s.FeedRun()
		time.Sleep(time.Duration(s.Cfg.Crawl.CrawlCycleMin) * time.Minute)
	}
}

func (s DefaultFeederService) FeedRun() {
	rootFolder := s.Cfg.Crawl.RootFolder
	s.Cfg.RunTime.CrawlRunNumber++
	logger.Info(fmt.Sprintf("Starting crawl run #%v...", s.Cfg.RunTime.CrawlRunNumber))
	err := s.crawlFolder(rootFolder, s.Cfg.Crawl.Extensions)
	if err != nil {
		logger.Error(fmt.Sprintf("Error crawling folder %v: ", rootFolder), err)
	}
	logger.Info(fmt.Sprintf("Finished crawl run #%v.", s.Cfg.RunTime.CrawlRunNumber))
	logger.Info(fmt.Sprintf("Number of entries in file list: %v", s.Repo.Size()))
	logger.Info("Starting to extract file data...")
	s.extractFileInfo()
	logger.Info("Finished extracting file data.")
}

func (s DefaultFeederService) crawlFolder(rootFolder string, extensions []string) error {
	var fi domain.FileInfo
	err := filepath.Walk(getTodayFolder(rootFolder),
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if misc.SliceContainsString(extensions, filepath.Ext(path)) {
				if s.Repo.Exists(path) {
					oldFile := s.Repo.GetFileData(path)
					if oldFile.FileInfo.ModTime() == info.ModTime() {
						logger.Info(fmt.Sprintf("File %v already exists and is unmodified. Not adding", path))
						return nil
					}
				}
				fi.FileInfo = info
				fi.Path = path
				fi.FromCalCMS = false
				fi.ScanTime = time.Now()
				fi.InfoExtracted = false
				s.Repo.Store(fi)
				logger.Info(fmt.Sprintf("File %v added", path))
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

	year := fmt.Sprintf("%d", time.Now().Year())
	month := fmt.Sprintf("%02d", time.Now().Month())
	day := fmt.Sprintf("%02d", time.Now().Day())

	return path.Join(rootFolder, year, month, day)

	//return path.Join(rootFolder, "2023", "12", "06")
}

func (s DefaultFeederService) extractFileInfo() error {
	var startTimeDisplay string
	folderExp := regexp.MustCompile(`[\\/]+(0[0-9]|1[0-9]|2[0-3])-(0[0-9]|[1-5][0-9])`)
	file1Exp := regexp.MustCompile(`([01][0-9]|2[0-3])[0-5][0-9]-([01][0-9]|2[0-3])[0-5][0-9]_`)
	file2Exp := regexp.MustCompile(`([01][0-9]|2[0-3])[0-5][0-9]_`)
	files := s.Repo.GetAll()
	for _, file := range *files {
		if !file.InfoExtracted {
			var timeData string
			var newInfo domain.FileInfo
			len, err := analyzeLength(file.Path, s.Cfg.Crawl.FfProbeTimeOut, s.Cfg.Crawl.FfprobePath)
			if err != nil {
				logger.Error("Could not analyze file length: ", err)
			}
			newInfo = file
			folderName := filepath.Dir(file.Path)
			fileName := filepath.Base(file.Path)
			switch {
			// Case 1: file has been uploaded via calCMS, start time is coded in folder name: "\HH-MM" or "/HH-MM"
			case folderExp.MatchString(folderName):
				{
					timeData = folderExp.FindString(folderName)
					newInfo.FromCalCMS = true
					newInfo.StartTime = timeData[1:3] + ":" + timeData[4:6]
				}
			// Case 2: file has been uploaded manually, time slot is coded in file name in the form "HHMM-HHMM_"
			case file1Exp.MatchString(fileName):
				{
					timeData = file1Exp.FindString(fileName)
					newInfo.StartTime = timeData[0:2] + ":" + timeData[2:4]
				}
			// Case 3: file has been uploaded manually, start time is coded in file name in the form "HHMM_"
			case file2Exp.MatchString(fileName):
				{
					timeData = file2Exp.FindString(fileName)
					newInfo.StartTime = timeData[0:2] + ":" + timeData[2:4]
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
