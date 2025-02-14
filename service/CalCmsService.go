// package service implements the services and their business logic that provide the main part of the program
package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
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

type CalCmsService interface {
	Query() error
}

// The calCms service handles all the communication with calCms and the necessary data transformation
type DefaultCalCmsService struct {
	Cfg  *config.AppConfig
	Repo *repositories.DefaultFileRepository
}

var (
	httpCalTr     http.Transport
	httpCalClient http.Client
	CalCmsPgm     struct {
		sync.RWMutex
		data domain.CalCmsPgmData
	}
)

// InitHttpCalClient sets the defaukt values for the http client used to query calCms
func InitHttpCalClient() {
	httpCalTr = http.Transport{
		DisableKeepAlives:  false,
		DisableCompression: false,
		MaxIdleConns:       0,
		IdleConnTimeout:    0,
	}
	httpCalClient = http.Client{
		Transport: &httpCalTr,
		Timeout:   5 * time.Second,
	}
}

// NewCalCmsService creates a new calCms service and injects its dependencies
func NewCalCmsService(cfg *config.AppConfig, repo *repositories.DefaultFileRepository) DefaultCalCmsService {
	InitHttpCalClient()
	return DefaultCalCmsService{
		Cfg:  cfg,
		Repo: repo,
	}
}

// insertData inserts new calCms data in a thread-safe manner
func (s DefaultCalCmsService) insertData(data domain.CalCmsPgmData) {
	CalCmsPgm.Lock()
	CalCmsPgm.data = data
	CalCmsPgm.Unlock()
}

// calcCalCmsEndDate calculates the end date based on a given start date used to query events from calCms
// this is used to query calCms for the day's events
func calcCalCmsEndDate(startDate string) (endDate string, e error) {
	sd, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return "", err
	}
	return sd.AddDate(0, 0, 1).Format("2006-01-02"), nil
}

// setCalCmsQueryState sets staus of last calCms interaction with result and time for status overview
func (s DefaultCalCmsService) setCalCmsQueryState(success bool) {
	s.Cfg.RunTime.Mu.Lock()
	defer s.Cfg.RunTime.Mu.Unlock()
	if success {
		s.Cfg.RunTime.LastCalCmsState = fmt.Sprintf("Succeeded (%v)", time.Now().Format("2006-01-02 15:04:05 -0700 MST"))
	} else {
		s.Cfg.RunTime.LastCalCmsState = fmt.Sprintf("Failed (%v)", time.Now().Format("2006-01-02 15:04:05 -0700 MST"))
	}

}

// getCalCmsEventData retrieves the today's event information from calCms
func (s DefaultCalCmsService) getCalCmsEventData() (eventData []byte, e error) {
	//API doc: https://github.com/rapilodev/racalmas/blob/master/docs/event-api.md
	//URL old: https://programm.coloradio.org/agenda/events.cgi?date=2024-04-09&template=event.json-p
	//URL new: https://programm.coloradio.org/agenda/events.cgi?from_date=2024-10-04&from_time=00:00&till_date=2024-10-05&till_time=00:00&template=event.json-p
	var (
		calCmsStartDate string
	)
	calUrl, err := url.Parse(s.Cfg.CalCms.CmsUrl)
	if err != nil {
		logger.Error("Cannot parse calCMS Url", err)
		return nil, err
	}
	query := url.Values{}
	calCmsStartDate = strings.ReplaceAll(helper.GetTodayFolder(s.Cfg.Misc.TestCrawl, s.Cfg.Misc.TestDate), "/", "-")
	calCmsEndDate, err := calcCalCmsEndDate(calCmsStartDate)
	if err != nil {
		return nil, err
	}
	query.Add("from_date", calCmsStartDate)
	query.Add("from_time", "00:00")
	query.Add("till_date", calCmsEndDate)
	query.Add("till_time", "00:00")
	query.Add("template", s.Cfg.CalCms.Template)
	calUrl.RawQuery = query.Encode()
	req, err := http.NewRequest("GET", calUrl.String(), nil)
	if err != nil {
		s.setCalCmsQueryState(false)
		logger.Error("Cannot build calCMS http request", err)
		return nil, err
	}
	resp, err := httpCalClient.Do(req)
	if err != nil {
		s.setCalCmsQueryState(false)
		logger.Error("Cannot execute calCMS http request", err)
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		s.setCalCmsQueryState(false)
		err := errors.New(resp.Status)
		logger.Errorf("Received status code %v from calCMS. %v", resp.StatusCode, err)
		return nil, err
	}
	defer resp.Body.Close()
	eventData, err = io.ReadAll(resp.Body)
	if err != nil {
		s.setCalCmsQueryState(false)
		logger.Error("Cannot read response data from calCMS", err)
		return nil, err
	}
	s.setCalCmsQueryState(true)
	return eventData, nil
}

// Query orchestrates the process of querying calCms and adding the retrieved information to the file representations in memory
func (s DefaultCalCmsService) Query() error {
	if s.Cfg.CalCms.QueryCalCms {
		logger.Info("Starting to add information from calCMS...")
		start := time.Now().UTC()
		data, err := s.getCalCmsEventData()
		if err != nil {
			logger.Error("error getting data from calCms", err)
			return err
		}
		CalCmsPgm.Lock()
		if err := json.Unmarshal(data, &CalCmsPgm.data); err != nil {
			logger.Error("Cannot convert calCMS response data to Json", err)
			return err
		}
		CalCmsPgm.Unlock()
		fc := s.EnrichFileInformation()
		end := time.Now().UTC()
		dur := end.Sub(start)
		logger.Infof("Added or updated information from calCMS for %v file(s), audio: %v, stream: %v (%v)", fc.TotalCount, fc.AudioCount, fc.StreamCount, dur.String())
		return nil
	}
	logger.Warn("calCMS query not enabled in configuration. Not querying")
	return nil

}

// EnrichFileInformation runs through all file representations and adds information from calCms where applicable
func (s DefaultCalCmsService) EnrichFileInformation() (fc dto.FileCounts) {
	folderDate := strings.ReplaceAll(helper.GetTodayFolder(s.Cfg.Misc.TestCrawl, s.Cfg.Misc.TestDate), "/", "-")
	if files := s.Repo.GetByDate(folderDate); files != nil {
		for _, file := range *files {
			if file.EventId != 0 {
				calCmsInfo, err := s.checkCalCmsEventData(file)
				if err != nil {
					logger.Errorf("Error while checking calCms event data for file %v: %v", file.Path, err)
					continue
				}
				newFile, nfc := mergeInfo(file, *calCmsInfo)
				fc.Add(nfc)
				if err := s.Repo.Store(newFile); err != nil {
					logger.Error("Error updating information in file repository", err)
				}
			}
		}
	}
	return fc
}

// mergeInfo combines the information from the existing file entry and data from calCms inot the new file entry
func mergeInfo(oldFileInfo domain.FileInfo, calCmsInfo dto.CalCmsEntry) (newFileInfo domain.FileInfo, fc dto.FileCounts) {
	newFileInfo = oldFileInfo
	if !oldFileInfo.FromCalCMS {
		logger.Warn("File not designated as \"From CalCMS\". This should not happen.")
		newFileInfo.FromCalCMS = true
	}
	if !oldFileInfo.StartTime.Equal(calCmsInfo.StartTime) {
		logger.Warnf("Start times differ. File: %v, calCMS: %v. Updating to value from calCMS.", oldFileInfo.StartTime, calCmsInfo.StartTime)
		newFileInfo.StartTime = calCmsInfo.StartTime
	}
	newFileInfo.EndTime = calCmsInfo.EndTime
	newFileInfo.CalCmsTitle = calCmsInfo.Title
	newFileInfo.CalCmsInfoExtracted = true
	if oldFileInfo.FileType == "Audio" {
		fc.AudioCount++
	}
	if (oldFileInfo.FileType == "Stream") && (oldFileInfo.StreamId != 0) {
		newFileInfo.Duration = float64(calCmsInfo.Duration.Seconds())
		fc.StreamCount++
	}
	if calCmsInfo.Live != 0 {
		newFileInfo.EventIsLive = true
	} else {
		newFileInfo.EventIsLive = false
	}
	fc.TotalCount++
	return
}

// checkCalCmsEventData evaluates calCms event data on a per file basis and performs some sanity checks
func (s DefaultCalCmsService) checkCalCmsEventData(file domain.FileInfo) (*dto.CalCmsEntry, error) {
	var (
		exportLive string
	)
	info, err := s.GetCalCmsEventDataForId(file.EventId)
	if err != nil {
		return nil, err
	}
	calCmsDate := strings.ReplaceAll(helper.GetTodayFolder(s.Cfg.Misc.TestCrawl, s.Cfg.Misc.TestDate), "/", "-")
	if (len(info) == 0) && (calCmsDate == file.FolderDate) {
		return nil, fmt.Errorf("no Id %v in calCMS", file.EventId)
	}
	if len(info) > 1 {
		logger.Warnf("Ambiguous information from calCMS. Found %v entries. Not adding information.", len(info))
		return nil, errors.New("multiple matches in calCMS")
	}
	if calCmsDate != file.FolderDate {
		return nil, fmt.Errorf("file has different date (%v) than calCms data (%v)", file.FolderDate, calCmsDate)
	}
	if (len(info) == 1) && (info[0].Live == 1) {
		if s.Cfg.Export.ExportLiveItems {
			exportLive = "Per configuration setting Live items will be exported."
		} else {
			exportLive = "Per configuration setting Live items will NOT be exported."
		}
		logger.Warnf("%v, Id: %v is designated as live, yet a file is present. %v", info[0].Title, info[0].EventId, exportLive)
		return &info[0], nil
	}
	return &info[0], nil
}

// GetCalCmsEntriesForHour retrieves all event data from the calCms data that start within a given hour
func (s DefaultCalCmsService) GetCalCmsEntriesForHour(hour string) (entries []dto.CalCmsEntry, e error) {
	CalCmsPgm.RLock()
	events := CalCmsPgm.data.Events
	CalCmsPgm.RUnlock()
	if len(events) > 0 {
		for _, event := range events {
			if (event.Live == 0) && (strings.HasPrefix(event.StartTimeName, hour)) {
				entry, err := s.convertEventToEntry(event)
				if err == nil {
					entries = append(entries, entry)
				}
			}
		}
	}
	return entries, nil
}

// GetCalCmsEventDataForId retrieves all event data from the calCms data for a given Event Id
func (s DefaultCalCmsService) GetCalCmsEventDataForId(id int) (entries []dto.CalCmsEntry, e error) {
	CalCmsPgm.RLock()
	events := CalCmsPgm.data.Events
	CalCmsPgm.RUnlock()
	if len(events) > 0 {
		for _, event := range events {
			if event.EventID == id {
				entry, err := s.convertEventToEntry(event)
				if err == nil {
					entries = append(entries, entry)
				} else {
					return entries, err
				}
			}
		}
	}
	return entries, nil
}

// convertEventToEntry converts calCms event data to its data transport object
func (s DefaultCalCmsService) convertEventToEntry(event domain.CalCmsEvent) (entry dto.CalCmsEntry, e error) {
	var (
		err1, err2 error
	)
	entry.Title = event.FullTitle
	entry.StartTime, err1 = time.ParseInLocation("2006-01-02T15:04:05", event.StartDatetime, time.Local)
	if err1 != nil {
		logger.Errorf("Could not parse %v into time. %v", event.StartDatetime, err1)
		return entry, err1
	}
	entry.EndTime, err2 = time.ParseInLocation("2006-01-02T15:04:05", event.EndDatetime, time.Local)
	if err2 != nil {
		logger.Errorf("Could not parse %v into time. %v", event.EndDatetime, err2)
		return entry, err2
	}
	entry.Duration = entry.EndTime.Sub(entry.StartTime)
	entry.EventId = event.EventID
	entry.Live = event.Live
	return entry, nil
}

// parseDuration is a helper function converting calCms duration into seconds for display purposes
func parseDuration(dur string) string {
	if len(dur) >= 8 {
		dStr := dur[0:2] + "h" + dur[3:5] + "m" + dur[6:8] + "s"
		d, err := time.ParseDuration(dStr)
		if err != nil {
			return "N/A"
		}
		return strconv.FormatFloat(math.Round(d.Seconds()/60), 'f', 1, 64)
	}
	return "N/A"
}

// extractFileInfo is a helper function that returns file status and file duration as strings
func extractFileInfo(files *domain.FileList, hashEnabled bool) (fileStatus string, duration string, fileSource string) {
	var fs string
	if len(*files) == 0 {
		return "N/A", "N/A", "N/A"
	}
	if len(*files) == 1 {
		if (*files)[0].FromCalCMS && (*files)[0].EventId != 0 {
			fs = "calCMS"
		} else {
			fs = "Manual"
		}
		return "Present", strconv.FormatFloat(math.Round((*files)[0].Duration/60), 'f', 1, 64), fs
	}
	if hashEnabled {
		filesIdentical, checksumAvail := checkHash(files)
		switch {
		case checksumAvail && filesIdentical:
			return "Multiple (identical)", strconv.FormatFloat(math.Round((*files)[0].Duration/60), 'f', 1, 64), "N/A"
		case checksumAvail && !filesIdentical:
			return "Multiple (different)", "N/A", "N/A"
		default:
			return "Multiple", "N/A", "N/A"
		}
	}
	return "Multiple", "N/A", "N/A"
}

// checkHash compares the has of all files and returns true, if the hash values of all files are identical
func checkHash(files *domain.FileList) (filesIdentical bool, checksumAvail bool) {
	var (
		hash string
	)
	if len(*files) < 2 {
		return false, false
	}
	filesIdentical = true
	for _, file := range *files {
		if file.Checksum == "" {
			return false, false
		} else {
			if hash == "" {
				hash = file.Checksum
			} else {
				filesIdentical = (hash == file.Checksum)
			}
		}
	}
	return filesIdentical, true
}

// convertEvent is a helper function that converts calCms data into the event representation
func (s DefaultCalCmsService) convertEvent(calCmsData domain.CalCmsPgmData) []dto.Event {
	var (
		el    []dto.Event
		files *domain.FileList
	)
	for _, event := range calCmsData.Events {
		if !misc.SliceContainsString(s.Cfg.CalCms.EventExclusion, event.Skey) {
			var ev dto.Event
			ev.EventId = strconv.Itoa(event.EventID)
			ev.StartDate = event.StartDate
			ev.StartTime = event.StartTime
			ev.EndTime = event.EndTime
			ev.Title = event.FullTitle
			ev.PlannedDuration = parseDuration(event.Duration)
			if event.Live == 0 {
				ev.EventType = "Preproduction"
			} else {
				ev.EventType = "Live"
			}
			if s.Cfg.CalCms.ShowNonCalCmsFiles {
				files = s.Repo.GetByIdAndHour(event.EventID, ev.StartTime[0:2], s.Cfg.Export.ExportLiveItems)
			} else {
				files = s.Repo.GetByEventId(event.EventID)
			}
			if files == nil {
				if event.Live == 0 {
					ev.FileStatus = "Missing"
				} else {
					ev.FileStatus = "N/A"
				}
				ev.ActualDuration = "N/A"
			} else {
				ev.FileStatus, ev.ActualDuration, ev.FileSource = extractFileInfo(files, s.Cfg.Crawl.GenerateHash)
			}
			el = append(el, ev)
		}
	}
	return el
}

// GetEvents orchestrates the generation of an event list for display on the web UI
func (s DefaultCalCmsService) GetEvents() ([]dto.Event, error) {
	var (
		calCmsData domain.CalCmsPgmData
	)
	if s.Cfg.CalCms.QueryCalCms {
		data, err := s.getCalCmsEventData()
		if err != nil {
			logger.Error("error getting data from calCms", err)
			return nil, err
		}
		if err := json.Unmarshal(data, &calCmsData); err != nil {
			logger.Error("Cannot convert calCMS response data to Json", err)
			return nil, err
		}
		el := s.convertEvent(calCmsData)
		s.countEvents(el)
		return el, nil
	} else {
		logger.Warn("calCMS query not enabled in configuration. Not querying.")
		return nil, nil
	}
}

// countEvents counts the events with their file status
func (s DefaultCalCmsService) countEvents(events []dto.Event) {
	var (
		presentCount, missingCount, multipleCount int
	)
	for _, ev := range events {
		switch {
		case ev.FileStatus == "Present":
			presentCount++
		case ev.FileStatus == "Missing":
			missingCount++
		case strings.Contains(ev.FileStatus, "Multiple"):
			multipleCount++
		}
	}
	s.Cfg.RunTime.Mu.Lock()
	defer s.Cfg.RunTime.Mu.Unlock()
	s.Cfg.RunTime.EventsPresent = presentCount
	s.Cfg.RunTime.EventsMissing = missingCount
	s.Cfg.RunTime.EventsMultiple = multipleCount
}

func (s DefaultCalCmsService) CountRun() {
	s.GetEvents()
}
