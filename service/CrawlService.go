package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
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
	Cfg    *config.AppConfig
	Repo   *repositories.DefaultFileRepository
	CalSvc CalCmsService
}

func NewCrawlService(cfg *config.AppConfig, repo *repositories.DefaultFileRepository, calSvc CalCmsService) DefaultCrawlService {
	return DefaultCrawlService{
		Cfg:    cfg,
		Repo:   repo,
		CalSvc: calSvc,
	}
}

func (s DefaultCrawlService) Crawl() {
	if s.Cfg.Crawl.RootFolder == "" {
		logger.Warn("No root folder given. Not running")
		return
	}
	if s.Cfg.RunTime.CrawlRunning {
		logger.Warn("Crawl already running. Not starting another one.")
	} else {
		s.Cfg.RunTime.CrawlRunning = true
		s.CrawlRun()
		if s.Cfg.CalCms.QueryCalCms {
			s.CalSvc.Query()
		}
		s.Cfg.RunTime.CrawlRunning = false
	}
}

func (s DefaultCrawlService) CrawlRun() {
	rootFolder := s.Cfg.Crawl.RootFolder
	s.Cfg.RunTime.CrawlRunNumber++
	s.Cfg.RunTime.LastCrawlDate = time.Now()
	logger.Info(fmt.Sprintf("Root folder: %v. Starting crawl #%v.", s.Cfg.Crawl.RootFolder, s.Cfg.RunTime.CrawlRunNumber))
	fileCount, err := s.crawlFolder(rootFolder, s.Cfg.Crawl.Extensions)
	if err != nil {
		logger.Error(fmt.Sprintf("Error crawling folder %v: ", rootFolder), err)
	}
	s.Cfg.RunTime.FilesInList = s.Repo.Size()
	logger.Info(fmt.Sprintf("Finished crawl run #%v. Added %v new files. %v files in list total.", s.Cfg.RunTime.CrawlRunNumber, fileCount, s.Cfg.RunTime.FilesInList))
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
	today := helper.GetTodayFolder(s.Cfg.Misc.TestCrawl, s.Cfg.Misc.TestDate)
	err := filepath.Walk(path.Join(rootFolder, today),
		func(srcPath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if misc.SliceContainsString(extensions, filepath.Ext(srcPath)) {
				if s.Repo.Exists(srcPath) {
					oldFile := s.Repo.Get(srcPath)
					if oldFile.ModTime == info.ModTime() {
						logger.Info(fmt.Sprintf("File %v already exists and is unmodified. Not adding", srcPath))
						return nil
					}
				}
				fi.ModTime = info.ModTime()
				fi.Path = srcPath
				fi.FromCalCMS = false
				fi.ScanTime = time.Now()
				rawFolder := strings.Trim(filepath.Dir(srcPath), rootFolder)[0:10]
				fi.FolderDate = strings.Replace(rawFolder, "\\", "-", -1)
				fi.InfoExtracted = false
				fi.EventId = s.parseEventId(srcPath)
				fileCount++
				s.Repo.Store(fi)
				if s.Cfg.Misc.TestCrawl {
					logger.Info(fmt.Sprintf("File %v added", srcPath))
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

func (s DefaultCrawlService) parseEventId(srcPath string) int {
	fileName := filepath.Base(srcPath)
	idExp := regexp.MustCompile(`-id\d+-`)
	if idExp.MatchString(fileName) {
		idRawStr := idExp.FindString(fileName)
		l := len(idRawStr)
		idRaw := idRawStr[3 : l-1]
		id, err := strconv.Atoi(idRaw)
		if err == nil {
			return id
		}
	}
	return 0
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
						newInfo.StartTime, _ = convertTime(timeData[1:3], timeData[4:6])
						newInfo.RuleMatched = "folder HH-MM (calCMS)"
					}
				// Condition: start time and end time is encoded in folder name: "/HHMM-HHMM"
				case folder2Exp.MatchString(folderName):
					{
						timeData = folder2Exp.FindString(folderName)
						timeData = strings.Replace(timeData, " ", "", -1)
						newInfo.FromCalCMS = true
						newInfo.StartTime, _ = convertTime(timeData[1:3], timeData[3:5])
						newInfo.EndTime, _ = convertTime(timeData[6:8], timeData[8:10])
						newInfo.RuleMatched = "folder HHMM-HHMM"
					}
				// Condition: start time (hour) and end time (hour) is encoded in folder name: "HH bis HH"
				case folder3Exp.MatchString(folderName):
					{
						timeData = folder3Exp.FindString(folderName)
						timeData = strings.Replace(timeData, " ", "", -1)
						newInfo.FromCalCMS = true
						newInfo.StartTime, _ = convertTime(timeData[1:3], "0")
						newInfo.EndTime, _ = convertTime(timeData[6:8], "0")
						newInfo.RuleMatched = "folder HH bis HH"
					}
				// Condition: start time and end time is encoded in file name in the form "HHMM-HHMM_"
				case file1Exp.MatchString(fileName):
					{
						timeData = file1Exp.FindString(fileName)
						timeData = strings.Replace(timeData, " ", "", -1)
						newInfo.StartTime, _ = convertTime(timeData[0:2], timeData[2:4])
						newInfo.EndTime, _ = convertTime(timeData[5:7], timeData[7:9])
						newInfo.RuleMatched = "file HHMM-HHMM"
					}
				// Condition: start time (hour) and end time (hour) is encoded in file name in the form "HH-HH_Uhr"
				case file2Exp.MatchString(fileName):
					{
						timeData = file2Exp.FindString(fileName)
						timeData = strings.Replace(timeData, " ", "", -1)
						newInfo.StartTime, _ = convertTime(timeData[0:2], "0")
						newInfo.EndTime, _ = convertTime(timeData[3:5], "0")
						newInfo.RuleMatched = "file HH-HH Uhr"
					}
				// Condition: only start time is encoded in the file name in the form of "HHMM_" (beware of date matching!)
				case file3Exp.MatchString(fileName):
					{
						timeData = file3Exp.FindString(fileName)
						newInfo.StartTime, _ = convertTime(timeData[0:2], timeData[2:4])
						newInfo.RuleMatched = "file HHMM_"
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
				if newInfo.StartTime.IsZero() {
					startTimeDisplay = "N/A"
				} else {
					startTimeDisplay = newInfo.StartTime.Format("15:04")
				}
				roundedDurationMin := math.Round(len / 60)
				logger.Info(fmt.Sprintf("Time Slot: % v, File: %v - Length (min): %v", startTimeDisplay, file.Path, roundedDurationMin))
			}
		}
	}
	return extractCount, nil
}

func convertTime(t1str string, t2str string) (time.Time, error) {
	t1, err := strconv.Atoi(t1str)
	if err != nil {
		logger.Error("converting error", err)
		return time.Time{}, err
	}
	t2, err := strconv.Atoi(t2str)
	if err != nil {
		logger.Error("converting error", err)
		return time.Time{}, err
	}
	time := helper.TimeFromHourAndMinute(t1, t2)
	return time, nil
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
