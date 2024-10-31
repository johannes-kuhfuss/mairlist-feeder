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
				t1 := time.Now()
				hash, err := generateHash(file.Path)
				t2 := time.Now()
				dur := t2.Sub(t1)
				if err != nil {
					logger.Errorf("Error when creating hash for %v. %v", file.Path, err)
				} else {
					file.Checksum = hash
					if err := s.Repo.Store(file); err != nil {
						logger.Error("Error storing file", err)
					} else {
						hashCount++
						logger.Infof("Added hash for file %v in %v seconds", file.Path, dur.Seconds())
					}
				}
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

// CrawlRun performs the crawling of the folder, the data enrichment and the has creation
func (s DefaultCrawlService) CrawlRun() {
	s.Cfg.RunTime.CrawlRunNumber++
	s.Cfg.RunTime.LastCrawlDate = time.Now()
	logger.Infof("Root folder: %v. Starting crawl #%v.", s.Cfg.Crawl.RootFolder, s.Cfg.RunTime.CrawlRunNumber)
	filesRemoved := s.checkForOrphanFiles()
	fileCount, err := s.crawlFolder(s.Cfg.Crawl.RootFolder, s.Cfg.Crawl.CrawlExtensions)
	if err != nil {
		logger.Errorf("Error crawling folder %v: . %v", s.Cfg.Crawl.RootFolder, err)
	}
	s.Cfg.RunTime.FilesInList = s.Repo.Size()
	logger.Infof("Finished crawl run #%v. Removed %v orphaned files. Added %v new files. %v files in list total.", s.Cfg.RunTime.CrawlRunNumber, filesRemoved, fileCount, s.Cfg.RunTime.FilesInList)
	if s.Repo.NewFiles() {
		logger.Info("Starting to extract file data...")
		fc, _ := s.extractFileInfo()
		logger.Infof("Finished extracting file data for %v files. %v audio files, %v stream files", fc.TotalCount, fc.AudioCount, fc.StreamCount)
		if s.Cfg.Crawl.GenerateHash {
			logger.Info("Starting to add hashes for new files...")
			hc := s.GenHashes()
			logger.Infof("Done adding hashes for %v new files.", hc)
		}
	} else {
		logger.Info("No (new) files in file list. No extraction needed.")
	}
	s.Cfg.RunTime.AudioFilesInList = s.Repo.AudioSize()
	s.Cfg.RunTime.StreamFilesInList = s.Repo.StreamSize()
}

// crawlFolder examines the files on disk and adds an entry in the in-memory representation
func (s DefaultCrawlService) crawlFolder(rootFolder string, crawlExtensions []string) (fileCount int, e error) {
	var (
		fi domain.FileInfo
	)
	today := helper.GetTodayFolder(s.Cfg.Misc.TestCrawl, s.Cfg.Misc.TestDate)
	err := filepath.WalkDir(path.Join(rootFolder, today),
		func(srcPath string, info fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if misc.SliceContainsString(crawlExtensions, filepath.Ext(srcPath)) {
				newFile, _ := info.Info()
				if s.Repo.Exists(srcPath) {
					oldFile := s.Repo.GetByPath(srcPath)
					if oldFile.ModTime == newFile.ModTime() {
						return nil
					}
				}
				fi.ModTime = newFile.ModTime()
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
					logger.Infof("File %v added", srcPath)
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
		startTimeDisplay   string
		roundedDurationMin float64
	)
	// /HH-MM (calCMS)
	folder1Exp := regexp.MustCompile(`[\\/]+([01][0-9]|2[0-3])-(0[0-9]|[1-5][0-9])`)
	// HHMM-HHMM
	file1Exp := regexp.MustCompile(`^([01][0-9]|2[0-3])(0[0-9]|[1-5][0-9])\s?-\s?([01][0-9]|2[0-3])(0[0-9]|[1-5][0-9])[_ -]`)
	// UL__HHMM-HHMM__ (upload tool)
	file2Exp := regexp.MustCompile(`^UL__([01][0-9]|2[0-3])(0[0-9]|[1-5][0-9])-([01][0-9]|2[0-3])(0[0-9]|[1-5][0-9])__`)
	if files := s.Repo.GetAll(); files != nil {
		for _, file := range *files {
			if !file.InfoExtracted {
				var timeData string
				var newInfo domain.FileInfo = file

				if helper.IsAudioFile(s.Cfg, file.Path) {
					newInfo.FileType = "Audio"
					techMd, err := analyzeTechMd(file.Path, s.Cfg.Crawl.FfProbeTimeOut, s.Cfg.Crawl.FfprobePath)
					if err != nil {
						logger.Error("Could not analyze file length: ", err)
					} else {
						newInfo.Duration = techMd.DurationSec
						newInfo.BitRate = techMd.BitRate
						newInfo.FormatName = techMd.FormatName
						roundedDurationMin = math.Round(techMd.DurationSec / 60)
						fc.AudioCount++
					}
				}
				if helper.IsStreamingFile(s.Cfg, file.Path) {
					newInfo.FileType = "Stream"
					name, id, err := analyzeStreamData(file.Path, s.Cfg.Crawl.StreamMap)
					if err != nil {
						logger.Error("Could not analyze stream data", err)
					} else {
						newInfo.StreamName = name
						newInfo.StreamId = id
						fc.StreamCount++
					}
				}
				folderName := filepath.Dir(file.Path)
				fileName := filepath.Base(file.Path)
				switch {
				// Condition: only start time is encoded in folder name: "/HH-MM" (calCMS)
				case folder1Exp.MatchString(folderName):
					{
						timeData = folder1Exp.FindString(folderName)
						newInfo.FromCalCMS = true
						newInfo.StartTime, _ = convertTime(timeData[1:3], timeData[4:6], file.FolderDate)
						newInfo.RuleMatched = "folder HH-MM (calCMS)"
					}
				// Condition: start time and end time is encoded in file name in the form "HHMM-HHMM_"
				case file1Exp.MatchString(fileName):
					{
						timeData = file1Exp.FindString(fileName)
						timeData = strings.Replace(timeData, " ", "", -1)
						newInfo.StartTime, _ = convertTime(timeData[0:2], timeData[2:4], file.FolderDate)
						newInfo.EndTime, _ = convertTime(timeData[5:7], timeData[7:9], file.FolderDate)
						newInfo.RuleMatched = "file HHMM-HHMM"
					}
				// Condition start time and end time is encoded in file name in the form "UL__HHMM-HHMM__" (upload tool)
				case file2Exp.MatchString(fileName):
					{
						timeData = file2Exp.FindString(fileName)
						newInfo.StartTime, _ = convertTime(timeData[4:6], timeData[6:8], file.FolderDate)
						newInfo.EndTime, _ = convertTime(timeData[9:11], timeData[11:13], file.FolderDate)
						newInfo.RuleMatched = "Upload Tool"
					}
				default:
					{
						newInfo.RuleMatched = "None"
					}
				}
				newInfo.InfoExtracted = true
				fc.TotalCount++
				if err := s.Repo.Store(newInfo); err != nil {
					logger.Error("Error while storing data: ", err)
				}
				if newInfo.StartTime.IsZero() {
					startTimeDisplay = "N/A"
				} else {
					startTimeDisplay = newInfo.StartTime.Format("15:04")
				}
				switch newInfo.FileType {
				case "Stream":
					logger.Infof("Time Slot: % v, File: %v (Stream Description)", startTimeDisplay, file.Path)
				default:
					logger.Infof("Time Slot: % v, File: %v - Length (min): %v", startTimeDisplay, file.Path, roundedDurationMin)
				}
			}
		}
	}
	return fc, nil
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
		logger.Error("Could not execute ffprobe: ", err)
		return nil, err
	}
	cancel()
	techMd, err := parseTechMd(outJson)
	if err != nil {
		logger.Error("Could not parse technical metadata from ffprobe: ", err)
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
