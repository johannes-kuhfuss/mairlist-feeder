// package service implements the services and their business logic that provide the main part of the program
package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"math"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/domain"
	"github.com/johannes-kuhfuss/mairlist-feeder/dto"
	"github.com/johannes-kuhfuss/mairlist-feeder/helper"
	"github.com/johannes-kuhfuss/mairlist-feeder/repositories"
	"github.com/johannes-kuhfuss/services_utils/logger"
	"github.com/johannes-kuhfuss/services_utils/misc"
)

type CrawlService interface {
	Crawl()
}

var (
	crmu sync.Mutex
)

// The crawl service handles the cyclical scanning of the supervised folder and the extraction and enrichment of data for all files
type DefaultCrawlService struct {
	Cfg    *config.AppConfig
	Repo   *repositories.DefaultFileRepository
	CalSvc CalCmsService
}

// NewCrawlService creates a new crawling service and injects its dependencies
func NewCrawlService(cfg *config.AppConfig, repo *repositories.DefaultFileRepository, calSvc CalCmsService) DefaultCrawlService {
	return DefaultCrawlService{
		Cfg:    cfg,
		Repo:   repo,
		CalSvc: calSvc,
	}
}

// Crawl orchestrates the crawling of the folder on disk
func (s DefaultCrawlService) Crawl() {
	if s.Cfg.Crawl.RootFolder == "" {
		logger.Warn("No root folder given. Not running")
		return
	}
	crmu.Lock()
	defer crmu.Unlock()
	s.Cfg.RunTime.CrawlRunning = true
	s.CrawlRun()
	s.Cfg.RunTime.CrawlRunning = false
	if s.Cfg.CalCms.QueryCalCms {
		s.CalSvc.Query()
	}
}

// GenHashes creates a hash for the files on disk to allow for easy checking for identical files
func (s DefaultCrawlService) GenHashes() (hashCount int) {
	if s.Repo.Size() > 0 {
		files := s.Repo.GetAll()
		for _, file := range *files {
			if file.Checksum == "" {
				hash, err := generateHash(file.Path)
				if err != nil {
					logger.Errorf("Error when creating hash for %v. %v", file.Path, err)
					return
				}
				file.Checksum = hash
				if err := s.Repo.Store(file); err != nil {
					logger.Error("Error storing file in repository", err)
					return
				}
				hashCount++
				logger.Infof("Added hash for file %v [%v]", file.Path, hash)
			}
		}
	}
	return
}

// checkForOrphanFiles removes files from the in-memory list, if they are no longer present on disk
func (s DefaultCrawlService) checkForOrphanFiles() (filesRemoved int) {
	if s.Repo.Size() > 0 {
		files := s.Repo.GetAll()
		for _, file := range *files {
			if _, err := os.Stat(file.Path); errors.Is(err, os.ErrNotExist) {
				if err := s.Repo.Delete(file.Path); err == nil {
					logger.Warnf("File %v not found on disk. Removing from list.", file.Path)
					filesRemoved++
				} else {
					logger.Error("Error removing orphaned file.", err)
				}
			}
		}
	}
	return
}

// CrawlRun performs the crawling of the folder, the data enrichment and the hash creation
func (s DefaultCrawlService) CrawlRun() {
	s.Cfg.RunTime.CrawlRunNumber++
	s.Cfg.RunTime.LastCrawlDate = time.Now()
	logger.Infof("Starting crawl run #%v (Root Folder: %v).", s.Cfg.RunTime.CrawlRunNumber, s.Cfg.Crawl.RootFolder)
	start := time.Now().UTC()
	filesRemoved := s.checkForOrphanFiles()
	fileCount, err := s.crawlFolder(s.Cfg.Crawl.RootFolder, s.Cfg.Crawl.CrawlExtensions)
	if err != nil {
		logger.Errorf("Error crawling folder %v: . %v", s.Cfg.Crawl.RootFolder, err)
	}
	ts := s.Repo.Size()
	end := time.Now().UTC()
	dur := end.Sub(start)
	logger.Infof("Finished crawl run #%v. Removed %v orphaned file(s). Added %v new file(s). %v file(s) in list total. (%v)", s.Cfg.RunTime.CrawlRunNumber, filesRemoved, fileCount, ts, dur.String())
	if s.Repo.NewFiles() {
		logger.Info("Starting to extract file data...")
		start = time.Now().UTC()
		fc, _ := s.extractFileInfo()
		end = time.Now().UTC()
		dur = end.Sub(start)
		logger.Infof("Extracted file data for %v file(s). %v audio file(s), %v stream file(s) (%v)", fc.TotalCount, fc.AudioCount, fc.StreamCount, dur.String())
		if s.Cfg.Crawl.GenerateHash {
			logger.Info("Starting to add hashes for new files...")
			start = time.Now().UTC()
			hc := s.GenHashes()
			end = time.Now().UTC()
			dur = end.Sub(start)
			logger.Infof("Added hashes for %v new file(s) (%v)", hc, dur.String())
		}
	} else {
		logger.Info("No (new) file(s) in file list. No extraction needed.")
	}
	as := s.Repo.AudioSize()
	es := s.Repo.StreamSize()
	s.Cfg.RunTime.Mu.Lock()
	defer s.Cfg.RunTime.Mu.Unlock()
	s.Cfg.RunTime.FilesInList = ts
	s.Cfg.RunTime.AudioFilesInList = as
	s.Cfg.RunTime.StreamFilesInList = es
}

// crawlFolder examines the files on disk and adds an entry in the in-memory representation
func (s DefaultCrawlService) crawlFolder(rootFolder string, crawlExtensions []string) (fileCount int, e error) {
	today := helper.GetTodayFolder(s.Cfg.Misc.TestCrawl, s.Cfg.Misc.TestDate)
	err := filepath.WalkDir(path.Join(rootFolder, today),
		func(srcPath string, info fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if misc.SliceContainsStringCI(crawlExtensions, filepath.Ext(srcPath)) {
				newFile, _ := info.Info()
				if s.Repo.Exists(srcPath) {
					oldFile := s.Repo.GetByPath(srcPath)
					if oldFile.ModTime == newFile.ModTime() {
						return nil
					}
					logger.Infof("Modification date changed. Updating %v", oldFile.Path)
				}
				fi := s.setNewFileData(newFile, srcPath, rootFolder)
				fileCount++
				if err := s.Repo.Store(fi); err != nil {
					logger.Error("Error while storing file in repository", err)
				}
			}
			return nil
		})
	if err != nil {
		return fileCount, err
	} else {
		return fileCount, nil
	}
}

// setNewFileData updates the file data with newly extracted values
func (s DefaultCrawlService) setNewFileData(newFile fs.FileInfo, srcPath string, rootFolder string) (fileInfo domain.FileInfo) {
	fileInfo.ModTime = newFile.ModTime()
	fileInfo.Path = srcPath
	fileInfo.FromCalCMS = false
	fileInfo.ScanTime = time.Now()
	rawFolder := strings.Trim(filepath.Dir(srcPath), rootFolder)[0:10]
	fileInfo.FolderDate = strings.Replace(rawFolder, "\\", "-", -1)
	fileInfo.InfoExtracted = false
	fileInfo.EventId = s.parseEventId(srcPath)
	return
}

// parseEventId is a helper function that determines the calCms event id from a file's file name
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

// generateHash generates an MD5 hash for a given file
func generateHash(path string) (hash string, e error) {
	hasher := md5.New()
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	_, err = hasher.Write(data)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// extractFileInfo determines the naming convention and implicitly the likely source of the file
// It also initiates the data extraction for technical metadata and streaming information
// The extracted information is stored in the file list in-memory
func (s DefaultCrawlService) extractFileInfo() (fc dto.FileCounts, e error) {
	var (
		exErr error
	)
	if files := s.Repo.GetAll(); files != nil {
		for _, file := range *files {
			if !file.InfoExtracted {
				var newInfo domain.FileInfo

				if helper.IsAudioFile(s.Cfg, file.Path) {
					newInfo, exErr = s.extractAudioInfo(file)
					fc.AudioCount++
				}
				if helper.IsStreamingFile(s.Cfg, file.Path) {
					newInfo = s.extractStreamInfo(file)
					fc.StreamCount++
				}
				newInfo = s.matchFolderName(newInfo)
				fc.TotalCount++
				if exErr == nil {
					newInfo.InfoExtracted = true
				}
				if err := s.Repo.Store(newInfo); err != nil {
					logger.Error("Error while storing file in repository", err)
				}
				logExtractResult(newInfo)
			}
		}
	}
	return fc, nil
}

// extractAudioInfo enriches the file information with audio file specific metadata
func (s DefaultCrawlService) extractAudioInfo(oldInfo domain.FileInfo) (newInfo domain.FileInfo, e error) {
	newInfo = oldInfo
	newInfo.FileType = "Audio"
	techMd, err := analyzeTechMd(oldInfo.Path, s.Cfg.Crawl.FfProbeTimeOut, s.Cfg.Crawl.FfprobePath)
	if err != nil {
		logger.Error("Could not analyze file length", err)
		return newInfo, err
	} else {
		newInfo.Duration = techMd.DurationSec
		newInfo.BitRate = techMd.BitRate
		newInfo.FormatName = techMd.FormatName
	}
	return newInfo, nil
}

// extractStreamInfo enriches the file information with stream file specific metadata
func (s DefaultCrawlService) extractStreamInfo(oldInfo domain.FileInfo) (newInfo domain.FileInfo) {
	newInfo = oldInfo
	newInfo.FileType = "Stream"
	name, id, err := analyzeStreamData(oldInfo.Path, s.Cfg.Crawl.StreamMap)
	if err != nil {
		logger.Error("Could not analyze stream data", err)
	} else {
		newInfo.StreamName = name
		newInfo.StreamId = id
	}
	return
}

// matchFolderName determines the source of the file, either calCms or naming convention
func (s DefaultCrawlService) matchFolderName(oldInfo domain.FileInfo) (newInfo domain.FileInfo) {
	var (
		timeData string
	)
	// /HH-MM (calCMS)
	folder1Exp := regexp.MustCompile(`[\\/]+([01][0-9]|2[0-3])-(0[0-9]|[1-5][0-9])`)
	// HHMM-HHMM
	file1Exp := regexp.MustCompile(`^([01][0-9]|2[0-3])(0[0-9]|[1-5][0-9])\s?-\s?([01][0-9]|2[0-3])(0[0-9]|[1-5][0-9])[_ -]`)
	newInfo = oldInfo
	folderName := filepath.Dir(oldInfo.Path)
	fileName := filepath.Base(oldInfo.Path)
	switch {
	// Condition: only start time is encoded in folder name: "/HH-MM" (calCMS)
	case folder1Exp.MatchString(folderName):
		{
			timeData = folder1Exp.FindString(folderName)
			newInfo.FromCalCMS = true
			newInfo.StartTime, _ = convertTime(timeData[1:3], timeData[4:6], oldInfo.FolderDate)
			newInfo.RuleMatched = "folder HH-MM (calCMS)"
		}
	// Condition: start time and end time is encoded in file name in the form "HHMM-HHMM_"
	case file1Exp.MatchString(fileName) && s.Cfg.Crawl.AddNonCalCmsFiles:
		{
			timeData = file1Exp.FindString(fileName)
			timeData = strings.Replace(timeData, " ", "", -1)
			newInfo.StartTime, _ = convertTime(timeData[0:2], timeData[2:4], oldInfo.FolderDate)
			newInfo.EndTime, _ = convertTime(timeData[5:7], timeData[7:9], oldInfo.FolderDate)
			newInfo.RuleMatched = "file HHMM-HHMM"
		}
	default:
		{
			newInfo.RuleMatched = "None"
		}
	}
	return
}

// setStartIimeDisplay formats the start stime for display purposes
func logExtractResult(fi domain.FileInfo) {
	var (
		startTimeDisplay string
	)
	if fi.StartTime.IsZero() {
		startTimeDisplay = "N/A"
	} else {
		startTimeDisplay = fi.StartTime.Format("15:04")
	}
	switch fi.FileType {
	case "Stream":
		logger.Infof("Time Slot: % v, File: %v (Stream Descriptor)", startTimeDisplay, fi.Path)
	default:
		roundedDurationMin := math.Round(fi.Duration / 60)
		logger.Infof("Time Slot: % v, File: %v - Length (min): %v", startTimeDisplay, fi.Path, roundedDurationMin)
	}

}

// convertTime is a helper function to convert time information extracted from the file names into a time.Time
func convertTime(t1str string, t2str string, folderDate string) (t time.Time, e error) {
	t1, err := strconv.Atoi(t1str)
	if err != nil {
		logger.Error("converting start time error", err)
		return time.Time{}, err
	}
	t2, err := strconv.Atoi(t2str)
	if err != nil {
		logger.Error("converting end time error", err)
		return time.Time{}, err
	}
	fd, err := time.Parse("2006-01-02", folderDate)
	if err != nil {
		logger.Error("converting folder date error", err)
		return time.Time{}, err
	}
	return helper.TimeFromHourAndMinuteAndDate(t1, t2, fd), nil
}

// analyzeTechMd runs ffprobe to extract technical metadata from audio files
func analyzeTechMd(essencePath string, timeout int, ffprobePath string) (techMetadata *dto.TechnicalMetadata, err error) {
	ctx := context.Background()
	timeoutDuration := time.Duration(timeout) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeoutDuration)
	// Syntax: ffprobe -show_format -print_format json -loglevel quiet <input_file>
	cmd := exec.CommandContext(ctx, ffprobePath, "-show_format", "-print_format", "json", "-loglevel", "quiet", essencePath)
	outJson, err := cmd.CombinedOutput()
	if err != nil {
		cancel()
		logger.Error("Could not execute ffprobe", err)
		return nil, err
	}
	cancel()
	techMd, err := parseTechMd(outJson)
	if err != nil {
		logger.Error("Could not parse technical metadata from ffprobe", err)
		return nil, err
	}
	return techMd, nil
}

// analyzeStreamData reads the file's contents to extract information about which stream is referred to
func analyzeStreamData(path string, streamMap map[string]int) (streamName string, streamId int, e error) {
	fileContents, err := os.ReadFile(path)
	if err != nil {
		logger.Error("Error reading stream description data from file", err)
		return "", 0, err
	}
	streamData := strings.ToLower(string(fileContents))
	for stream, id := range streamMap {
		if strings.Contains(streamData, stream) {
			return stream, id, nil
		}
	}
	return "", 0, errors.New("no such stream configured")
}

// parseTechMd interprets the output of ffprobe and extracts the desired technical metadata
func parseTechMd(ffprobedata []byte) (techMetadata *dto.TechnicalMetadata, err error) {
	var (
		result domain.FfprobeResult
		techMd dto.TechnicalMetadata
	)
	if err := json.Unmarshal(ffprobedata, &result); err != nil {
		return nil, err
	}
	durFloat, err := strconv.ParseFloat(result.Format.Duration, 64)
	if err != nil {
		return nil, err
	}
	bitRateRaw, _ := strconv.ParseFloat(result.Format.BitRate, 64)
	techMd.DurationSec = durFloat
	techMd.BitRate = int64(math.Round(bitRateRaw / float64(1024)))
	techMd.FormatName = result.Format.FormatLongName
	return &techMd, nil
}
