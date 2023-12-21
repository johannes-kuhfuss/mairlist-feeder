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
	Cfg   *config.AppConfig
	Store *repositories.DefaultFileRepository
}

var (
	fileList domain.FileList
)

func NewFeederService(cfg *config.AppConfig, store *repositories.DefaultFileRepository) DefaultFeederService {
	return DefaultFeederService{
		Cfg:   cfg,
		Store: store,
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
	logger.Info(fmt.Sprintf("Number of entries in file list: %v", len(fileList)))
	folderExp := regexp.MustCompile(`[\\/]+(0[0-9]|1[0-9]|2[0-3])-(0[0-9]|[1-5][0-9])`)
	file1Exp := regexp.MustCompile(`([01][0-9]|2[0-3])[0-5][0-9]-([01][0-9]|2[0-3])[0-5][0-9]_`)
	file2Exp := regexp.MustCompile(`([01][0-9]|2[0-3])[0-5][0-9]_`)
	for idx, file := range fileList {
		var timeData string
		len, err := analyzeLength(file.Path, s.Cfg.MAirList.FfProbeTimeOut, s.Cfg.MAirList.FfprobePath)
		if err != nil {
			logger.Error("Could not analyze file length: ", err)
		}
		fileList[idx].Duration = len
		folderName := filepath.Dir(file.Path)
		fileName := filepath.Base(file.Path)
		switch {
		// Case 1: file has been uploaded via calCMS, start time is coded in folder name: "\HH-MM" or "/HH-MM"
		case folderExp.MatchString(folderName):
			{
				timeData = folderExp.FindString(folderName)
				fileList[idx].FromCalCMS = true
				fileList[idx].StartTime = timeData[1:3] + ":" + timeData[4:6]
			}
		// Case 2: file has been uploaded manually, time slot is coded in file name in the form "HHMM-HHMM_"
		case file1Exp.MatchString(fileName):
			{
				timeData = file1Exp.FindString(fileName)
				fileList[idx].StartTime = timeData[0:2] + ":" + timeData[2:4]
			}
		// Case 3: file has been uploaded manually, start time is coded in file name in the form "HHMM_"
		case file2Exp.MatchString(fileName):
			{
				timeData = file2Exp.FindString(fileName)
				fileList[idx].StartTime = timeData[0:2] + ":" + timeData[2:4]
			}
		}
		logger.Info(fmt.Sprintf("Time Slot: % v, File: %v - Length (sec): %v", fileList[idx].StartTime, file.Path, len))
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
				fi.FileInfo = info
				fi.Path = path
				fi.FromCalCMS = false
				fi.ScanTime = time.Now()
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

func getTodayFolder(rootFolder string) string {
	/*
		year := fmt.Sprintf("%d", time.Now().Year())
		month := fmt.Sprintf("%02d", time.Now().Month())
		day := fmt.Sprintf("%02d", time.Now().Day())

		return path.Join(rootFolder, year, month, day)
	*/
	return path.Join(rootFolder, "2023", "12", "06")
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
