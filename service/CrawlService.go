// package service implements the services and their business logic that provide the main part of the program
package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/johannes-kuhfuss/mairlist-feeder/appstate"
	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/domain"
	"github.com/johannes-kuhfuss/mairlist-feeder/dto"
	"github.com/johannes-kuhfuss/mairlist-feeder/helper"
	"github.com/johannes-kuhfuss/mairlist-feeder/repositories"
	"github.com/johannes-kuhfuss/services_utils/logger"
)

type Crawler interface {
	Crawl() error
}

var (
	idExp      = regexp.MustCompile(`-id\d+-`)
	folder1Exp = regexp.MustCompile(`[\\/]+([01][0-9]|2[0-3])-(0[0-9]|[1-5][0-9])`)
	file1Exp   = regexp.MustCompile(`^([01][0-9]|2[0-3])(0[0-9]|[1-5][0-9])\s?-\s?([01][0-9]|2[0-3])(0[0-9]|[1-5][0-9])[_ -]`)
)

// The crawl service handles the cyclical scanning of the supervised folder and the extraction and enrichment of data for all files
type DefaultCrawlService struct {
	Cfg    *config.AppConfig
	State  *appstate.AppState
	Repo   *repositories.DefaultFileRepository
	CalSvc CalCmsQuerier
	Now    func() time.Time
	RunCmd func(context.Context, string, ...string) ([]byte, error)
	mu     *sync.Mutex
}

// NewCrawlService creates a new crawling service and injects its dependencies
func NewCrawlService(cfg *config.AppConfig, repo *repositories.DefaultFileRepository, calSvc CalCmsQuerier) DefaultCrawlService {
	return NewCrawlServiceWithState(cfg, appstate.New(), repo, calSvc)
}

func NewCrawlServiceWithState(cfg *config.AppConfig, state *appstate.AppState, repo *repositories.DefaultFileRepository, calSvc CalCmsQuerier) DefaultCrawlService {
	return DefaultCrawlService{
		Cfg:    cfg,
		State:  state,
		Repo:   repo,
		CalSvc: calSvc,
		Now:    time.Now,
		RunCmd: runCommand,
		mu:     &sync.Mutex{},
	}
}

// Crawl orchestrates the crawling of the folder on disk
func (s DefaultCrawlService) Crawl() (err error) {
	start := s.Now()
	defer func() {
		recordRunMetrics(s.State, "crawl", start, err)
	}()
	if s.Cfg.Crawl.RootFolder == "" {
		err = errors.New("no root folder given")
		logger.Warn("No root folder given. Not running")
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.State.Runtime.Mu.Lock()
	s.State.Runtime.CrawlRunning = true
	s.State.Runtime.Mu.Unlock()
	defer func() {
		s.State.Runtime.Mu.Lock()
		s.State.Runtime.CrawlRunning = false
		s.State.Runtime.Mu.Unlock()
	}()
	err = s.CrawlRun()
	if s.Cfg.CalCms.QueryCalCms && s.CalSvc != nil {
		err = errors.Join(err, s.CalSvc.Query())
	}
	return err
}

// GenHashes creates a hash for the files on disk to allow for easy checking for identical files
func (s DefaultCrawlService) GenHashes() (hashCount int, e error) {
	if s.Repo.Size() > 0 {
		files := s.Repo.GetAll()
		for _, file := range *files {
			if file.Checksum == "" {
				hash, err := generateHash(file.Path)
				if err != nil {
					logger.Errorf("Error when creating hash for %v. %v", file.Path, err)
					return hashCount, err
				}
				file.Checksum = hash
				if err := s.Repo.Store(file); err != nil {
					logger.Error("Error storing file in repository", err)
					return hashCount, err
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
func (s DefaultCrawlService) CrawlRun() error {
	var (
		crawlDur, extractDur, hashDur time.Duration
		runErr                        error
	)
	s.State.Runtime.Mu.Lock()
	sinceLastCrawl := time.Since(s.State.Runtime.LastCrawlDate)
	s.State.Runtime.CrawlRunNumber++
	s.State.Runtime.LastCrawlDate = s.Now()
	crawlRunNumber := s.State.Runtime.CrawlRunNumber
	s.State.Runtime.Mu.Unlock()
	s.State.Metrics.SetCrawlInterval("sincelastcrawl", sinceLastCrawl.Seconds())

	logger.Infof("Starting crawl run #%v (Root Folder: %v). Time since last crawl: %v", crawlRunNumber, s.Cfg.Crawl.RootFolder, sinceLastCrawl)
	start := time.Now().UTC()
	filesRemoved := s.checkForOrphanFiles()
	fileCount := 0
	for _, crawlDate := range helper.GetCrawlDates(s.Cfg.Misc.TestCrawl, s.Cfg.Misc.TestDate) {
		fc, err := s.crawlFolderForDate(s.Cfg.Crawl.RootFolder, s.Cfg.Crawl.CrawlExtensions, crawlDate)
		fileCount += fc
		if err != nil {
			logger.Errorf("Error crawling folder %v for date %v: %v", s.Cfg.Crawl.RootFolder, domain.FormatFolderDate(crawlDate), err)
			runErr = errors.Join(runErr, err)
		}
	}
	ts := s.Repo.Size()
	end := time.Now().UTC()
	crawlDur = end.Sub(start)
	s.State.Metrics.ObserveFastEvent("lastcrawl", crawlDur.Seconds())
	logger.Infof("Finished crawl run #%v. Removed %v orphaned file(s). Added %v new file(s). %v file(s) in list total. (%v)", crawlRunNumber, filesRemoved, fileCount, ts, crawlDur.String())
	if s.Repo.NewFiles() {
		logger.Info("Starting to extract file data...")
		start = time.Now().UTC()
		fc, err := s.extractFileInfo()
		if err != nil {
			runErr = errors.Join(runErr, err)
		}
		end = time.Now().UTC()
		extractDur = end.Sub(start)
		s.State.Metrics.ObserveFastEvent("lastextraction", extractDur.Seconds())
		logger.Infof("Extracted file data for %v file(s). %v audio file(s), %v stream file(s) (%v)", fc.TotalCount, fc.AudioCount, fc.StreamCount, extractDur.String())
		if s.Cfg.Crawl.GenerateHash {
			logger.Info("Starting to add hashes for new files...")
			start = time.Now().UTC()
			hc, err := s.GenHashes()
			if err != nil {
				runErr = errors.Join(runErr, err)
			}
			end = time.Now().UTC()
			hashDur = end.Sub(start)
			s.State.Metrics.ObserveFastEvent("lasthash", hashDur.Seconds())
			logger.Infof("Added hashes for %v new file(s) (%v)", hc, hashDur.String())
		}
	} else {
		logger.Info("No (new) file(s) in file list. No extraction needed.")
	}
	as := s.Repo.AudioSize()
	es := s.Repo.StreamSize()
	s.State.Metrics.SetFileNumber("total", float64(ts))
	s.State.Metrics.SetFileNumber("audio", float64(as))
	s.State.Metrics.SetFileNumber("stream", float64(es))
	return runErr
}

// crawlFolder examines the files on disk and adds an entry in the in-memory representation
func (s DefaultCrawlService) crawlFolder(rootFolder string, crawlExtensions []string) (fileCount int, e error) {
	return s.crawlFolderForDate(rootFolder, crawlExtensions, helper.DateForFolder(s.Cfg.Misc.TestCrawl, s.Cfg.Misc.TestDate, 0))
}

// crawlFolderForDate examines one dated folder on disk and adds entries to the in-memory representation.
func (s DefaultCrawlService) crawlFolderForDate(rootFolder string, crawlExtensions []string, folderDate time.Time) (fileCount int, e error) {
	folder := helper.FolderForDate(folderDate)
	folderPath := filepath.Join(rootFolder, folder)
	if _, err := os.Stat(folderPath); errors.Is(err, os.ErrNotExist) {
		logger.Infof("Crawl folder %v does not exist. Skipping.", folderPath)
		return 0, nil
	} else if err != nil {
		return 0, err
	}
	err := filepath.WalkDir(folderPath,
		func(srcPath string, info fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if slices.ContainsFunc(crawlExtensions, func(s string) bool { return strings.EqualFold(s, filepath.Ext(srcPath)) }) {
				newFile, err := info.Info()
				if err != nil {
					return err
				}
				if s.Repo.Exists(srcPath) {
					oldFile := s.Repo.GetByPath(srcPath)
					if time.Time.Equal(oldFile.ModTime, newFile.ModTime()) {
						return nil
					}
					logger.Infof("Modification date changed. Updating %v", oldFile.Path)
				} else {
					fileCount++
				}
				fi, err := s.setNewFileData(newFile, srcPath, rootFolder)
				if err != nil {
					return err
				}
				if err := s.Repo.Store(fi); err != nil {
					return fmt.Errorf("storing file %q in repository: %w", srcPath, err)
				}
			}
			return nil
		})
	return fileCount, err
}

// setNewFileData updates the file data with newly extracted values
func (s DefaultCrawlService) setNewFileData(newFile fs.FileInfo, srcPath string, rootFolder string) (fileInfo domain.FileInfo, e error) {
	fileInfo.ModTime = newFile.ModTime()
	fileInfo.Path = srcPath
	fileInfo.FromCalCMS = false
	fileInfo.ScanTime = s.Now()
	folderDate, err := folderDateFromPath(srcPath, rootFolder)
	if err != nil {
		return domain.FileInfo{}, err
	}
	fileInfo.FolderDate = folderDate
	fileInfo.InfoExtracted = false
	fileInfo.EventId = s.parseEventId(srcPath)
	return fileInfo, nil
}

// parseEventId is a helper function that determines the calCms event id from a file's file name
func (s DefaultCrawlService) parseEventId(srcPath string) int {
	fileName := filepath.Base(srcPath)
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

// folderDateFromPath extracts the YYYY-MM-DD folder date below the crawl root.
func folderDateFromPath(srcPath, rootFolder string) (time.Time, error) {
	relDir, err := filepath.Rel(rootFolder, filepath.Dir(srcPath))
	if err != nil {
		return time.Time{}, err
	}
	parts := strings.Split(filepath.ToSlash(relDir), "/")
	if len(parts) < 3 {
		return time.Time{}, fmt.Errorf("path %q does not contain a YYYY/MM/DD folder below root %q", srcPath, rootFolder)
	}
	folderDate, err := domain.ParseFolderDate(strings.Join(parts[:3], "-"))
	if err != nil {
		return time.Time{}, fmt.Errorf("path %q does not contain a valid YYYY/MM/DD folder below root %q: %w", srcPath, rootFolder, err)
	}
	return folderDate, nil
}

// generateHash generates an MD5 hash for a given file for duplicate detection, not for security.
func generateHash(path string) (hash string, e error) {
	hasher := md5.New()
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	if _, err = io.Copy(hasher, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// extractFileInfo determines the naming convention and implicitly the likely source of the file
// It also initiates the data extraction for technical metadata and streaming information
// The extracted information is stored in the file list in-memory
func (s DefaultCrawlService) extractFileInfo() (fc dto.FileCounts, e error) {
	if files := s.Repo.GetAll(); files != nil {
		for _, file := range *files {
			if !file.InfoExtracted {
				var exErr error
				newInfo := file

				if helper.IsAudioFile(s.Cfg, file.Path) {
					newInfo, exErr = s.extractAudioInfo(file)
					fc.AudioCount++
				}
				if helper.IsStreamingFile(s.Cfg, file.Path) {
					var streamErr error
					newInfo, streamErr = s.extractStreamInfo(file)
					exErr = errors.Join(exErr, streamErr)
					fc.StreamCount++
				}
				newInfo = s.matchFolderName(newInfo)
				fc.TotalCount++
				if exErr == nil {
					newInfo.InfoExtracted = true
				} else {
					e = errors.Join(e, fmt.Errorf("extracting file info for %q: %w", file.Path, exErr))
				}
				if err := s.Repo.Store(newInfo); err != nil {
					logger.Error("Error while storing file in repository", err)
					e = errors.Join(e, err)
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
	newInfo.FileType = domain.FileTypeAudio
	techMd, err := analyzeTechMdWithRunner(oldInfo.Path, s.Cfg.Crawl.FfProbeTimeOut, s.Cfg.Crawl.FfprobePath, s.RunCmd)
	if err != nil {
		logger.Error("Could not analyze file length", err)
		return newInfo, err
	} else {
		newInfo.Duration = techMd.Duration
		newInfo.BitRate = techMd.BitRate
		newInfo.FormatName = techMd.FormatName
	}
	return newInfo, nil
}

// extractStreamInfo enriches the file information with stream file specific metadata
func (s DefaultCrawlService) extractStreamInfo(oldInfo domain.FileInfo) (newInfo domain.FileInfo, e error) {
	newInfo = oldInfo
	newInfo.FileType = domain.FileTypeStream
	name, id, err := analyzeStreamData(oldInfo.Path, s.Cfg.Crawl.StreamMap)
	if err != nil {
		logger.Error("Could not analyze stream data", err)
		return newInfo, err
	} else {
		newInfo.StreamName = name
		newInfo.StreamId = id
	}
	return newInfo, nil
}

// matchFolderName determines the source of the file, either calCms or naming convention
func (s DefaultCrawlService) matchFolderName(oldInfo domain.FileInfo) (newInfo domain.FileInfo) {
	var (
		timeData string
	)
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

// logExtractResult logs the extracted file information.
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
	case domain.FileTypeStream:
		logger.Infof("Time Slot: % v, File: %v (Stream Descriptor)", startTimeDisplay, fi.Path)
	default:
		roundedDurationMin := math.Round(fi.Duration.Minutes())
		logger.Infof("Time Slot: % v, File: %v - Length (min): %v", startTimeDisplay, fi.Path, roundedDurationMin)
	}

}

// convertTime is a helper function to convert time information extracted from the file names into a time.Time
func convertTime(t1str, t2str string, folderDate time.Time) (t time.Time, e error) {
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
	return helper.TimeFromHourAndMinuteAndDate(t1, t2, folderDate), nil
}

// analyzeTechMd runs ffprobe to extract technical metadata from audio files
func analyzeTechMd(essencePath string, timeout int, ffprobePath string) (techMetadata *dto.TechnicalMetadata, err error) {
	return analyzeTechMdWithRunner(essencePath, timeout, ffprobePath, runCommand)
}

func analyzeTechMdWithRunner(essencePath string, timeout int, ffprobePath string, runner func(context.Context, string, ...string) ([]byte, error)) (techMetadata *dto.TechnicalMetadata, err error) {
	ctx := context.Background()
	timeoutDuration := time.Duration(timeout) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeoutDuration)
	defer cancel()
	// Syntax: ffprobe -show_format -print_format json -loglevel quiet <input_file>
	outJson, err := runner(ctx, ffprobePath, "-show_format", "-print_format", "json", "-loglevel", "quiet", essencePath)
	if err != nil {
		logger.Error("Could not execute ffprobe", err)
		return nil, err
	}
	techMd, err := parseTechMd(outJson)
	if err != nil {
		logger.Error("Could not parse technical metadata from ffprobe", err)
		return nil, err
	}
	return techMd, nil
}

func runCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.CombinedOutput()
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
		if strings.Contains(streamData, strings.ToLower(stream)) {
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
	techMd.Duration = time.Duration(durFloat * float64(time.Second))
	techMd.BitRate = int64(math.Round(bitRateRaw / float64(1024)))
	techMd.FormatName = result.Format.FormatLongName
	return &techMd, nil
}
