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
	"time"

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
	for _, file := range rawFileList {
		folder := filepath.Dir(file.FilePath)
		if exp.MatchString(folder) {
			timePos := exp.FindString(folder)
			len, err := analyzeLength(file.FilePath, s.Cfg.MAirList.FfProbeTimeOut, s.Cfg.MAirList.FfprobePath)
			if err != nil {
				logger.Error("Could not analyze file length: ", err)
			}
			logger.Info(fmt.Sprintf("Time Slot: % v, File: %v - Length (sec): %v", timePos, file.FilePath, len))
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

	year := fmt.Sprintf("%d", time.Now().Year())
	month := fmt.Sprintf("%02d", time.Now().Month())
	day := fmt.Sprintf("%02d", time.Now().Day())

	return path.Join(rootFolder, year, month, day)

	//return path.Join(rootFolder, "2023", "12")
}

func analyzeLength(path string, timeout int, ffprobe string) (len int, err error) {
	ctx := context.Background()
	timeoutDuration := time.Duration(timeout) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeoutDuration)
	// ffprobe -show_format -print_format json -loglevel quiet <input_file>
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

func parseDuration(ffprobedata []byte) (durationSec int, err error) {
	var result domain.FfprobeResult
	err = json.Unmarshal(ffprobedata, &result)
	if err != nil {
		return 0, err
	}
	durFloat, err := strconv.ParseFloat(result.Format.Duration, 64)
	if err != nil {
		return 0, err
	}
	return int(math.Round(durFloat)), nil
}
