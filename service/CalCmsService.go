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

// NewCalCmsService creates a new calCms service and injects dependencies
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

// calcCalCmsEndDate calculates the end date based on a gioven start date used to query events from calCms
func calcCalCmsEndDate(startDate string) (endDate string, e error) {
	d, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return "", err
	}
	endDate = d.AddDate(0, 0, 1).Format("2006-01-02")
	return endDate, nil
}

// getCalCmsData retrieves the today's event information from calCms
func (s DefaultCalCmsService) getCalCmsData() (data []byte, e error) {
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
		s.Cfg.RunTime.LastCalCmsState = fmt.Sprintf("Failed (%v)", time.Now().Format("2006-01-02 15:04:05 -0700 MST"))
		logger.Error("Cannot build calCMS http request", err)
		return nil, err
	}
	resp, err := httpCalClient.Do(req)
	if err != nil {
		s.Cfg.RunTime.LastCalCmsState = fmt.Sprintf("Failed (%v)", time.Now().Format("2006-01-02 15:04:05 -0700 MST"))
		logger.Error("Cannot execute calCMS http request", err)
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		s.Cfg.RunTime.LastCalCmsState = fmt.Sprintf("Failed (%v)", time.Now().Format("2006-01-02 15:04:05 -0700 MST"))
		err := errors.New(resp.Status)
		logger.Error(fmt.Sprintf("Received status code %v from calCMS", resp.StatusCode), err)
		return nil, err
	}
	defer resp.Body.Close()
	data, err = io.ReadAll(resp.Body)
	if err != nil {
		s.Cfg.RunTime.LastCalCmsState = fmt.Sprintf("Failed (%v)", time.Now().Format("2006-01-02 15:04:05 -0700 MST"))
		logger.Error("Cannot read response data from calCMS http request", err)
		return nil, err
	}
	s.Cfg.RunTime.LastCalCmsState = fmt.Sprintf("Succeeded (%v)", time.Now().Format("2006-01-02 15:04:05 -0700 MST"))
	return data, nil
}

// Query orchestrates the process of querying calCms and adding the retrieved information to the file representations in memory
func (s DefaultCalCmsService) Query() error {
	if s.Cfg.CalCms.QueryCalCms {
		logger.Info("Starting to add information from calCMS...")
		data, err := s.getCalCmsData()
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
		logger.Info(fmt.Sprintf("Added / updated information from calCMS for %v files, audio: %v, stream: %v", fc.TotalCount, fc.AudioCount, fc.StreamCount))
		return nil
	} else {
		logger.Warn("calCMS query not enabled in configuration. Not querying.")
		return nil
	}
}

// EnrichFileInformation runs through all file representations and adds information from calCms wherer applicable
func (s DefaultCalCmsService) EnrichFileInformation() dto.FileCounts {
	var (
		newFile domain.FileInfo
		fc      dto.FileCounts
	)
	if files := s.Repo.GetAll(); files != nil {
		for _, file := range *files {
			if file.EventId != 0 {
				info, err := s.checkCalCmsData(file)
				if err != nil {
					continue
				}
				newFile = file
				if !file.FromCalCMS {
					logger.Warn("File not designated as \"From CalCMS\". This should not happen.")
					newFile.FromCalCMS = true
				}
				if !file.StartTime.Equal(info.StartTime) {
					logger.Warn(fmt.Sprintf("Start times differ. File: %v, calCMS: %v. Updating to value from calCMS.", file.StartTime, info.StartTime))
					newFile.StartTime = info.StartTime
				}
				newFile.EndTime = info.EndTime
				newFile.CalCmsTitle = info.Title
				newFile.CalCmsInfoExtracted = true
				if file.FileType == "Audio" {
					fc.AudioCount++
				}
				if (file.FileType == "Stream") && (file.StreamId != 0) {
					newFile.Duration = float64(info.Duration.Seconds())
					fc.StreamCount++
				}
				fc.TotalCount++
				if err := s.Repo.Store(newFile); err != nil {
					logger.Error("Error updating information in file repository", err)
				}
			}
		}
	}
	return fc
}

// checkCalCmsData evaluates calCms event data on a per file basis and performs some sanity checks
func (s DefaultCalCmsService) checkCalCmsData(file domain.FileInfo) (*dto.CalCmsEntry, error) {
	info, err := s.GetCalCmsDataForId(file.EventId)
	if err != nil {
		logger.Error("Error retrieving calCMS info: ", err)
		return nil, err
	}
	calCmsDate := strings.ReplaceAll(helper.GetTodayFolder(s.Cfg.Misc.TestCrawl, s.Cfg.Misc.TestDate), "/", "-")
	if (len(info) == 0) && (calCmsDate == file.FolderDate) {
		logger.Warn(fmt.Sprintf("No information from calCMS for Id %v in today's calCMS events", file.EventId))
		return nil, errors.New("no such id in calCMS")
	}
	if len(info) > 1 {
		logger.Warn(fmt.Sprintf("Ambiguous information from calCMS. Found %v entries. Not adding information.", len(info)))
		return nil, errors.New("multiple matches in calCMS")
	}
	if (len(info) == 1) && (info[0].Live == 1) {
		logger.Warn(fmt.Sprintf("%v, Id: %v is designated as live. Not adding information.", info[0].Title, info[0].EventId))
		return nil, errors.New("event is live in calCMS")
	}
	if calCmsDate != file.FolderDate {
		return nil, errors.New("file has different date from calCmsData")
	}
	return &info[0], nil
}

// GetCalCmsDataForHour retrieves all event data from the calCms data that start within a given hour
func (s DefaultCalCmsService) GetCalCmsDataForHour(hour string) ([]dto.CalCmsEntry, error) {
	var entries []dto.CalCmsEntry
	CalCmsPgm.RLock()
	events := CalCmsPgm.data.Events
	CalCmsPgm.RUnlock()
	if len(events) > 0 {
		for _, event := range events {
			if (event.Live == 0) && (strings.HasPrefix(event.StartTimeName, hour)) {
				entry, err := s.convertToEntry(event)
				if err == nil {
					entries = append(entries, entry)
				}
			}
		}
	}
	return entries, nil
}

// GetCalCmsDataForId retrieves all event data from the calCms data for a given Event Id
func (s DefaultCalCmsService) GetCalCmsDataForId(id int) ([]dto.CalCmsEntry, error) {
	var entries []dto.CalCmsEntry
	CalCmsPgm.RLock()
	events := CalCmsPgm.data.Events
	CalCmsPgm.RUnlock()
	if len(events) > 0 {
		for _, event := range events {
			if event.EventID == id {
				entry, err := s.convertToEntry(event)
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

// convertToEntry converts calCms event data to its data transport object
func (s DefaultCalCmsService) convertToEntry(event domain.CalCmsEvent) (dto.CalCmsEntry, error) {
	var (
		entry      dto.CalCmsEntry
		err1, err2 error
	)
	entry.Title = event.FullTitle
	entry.StartTime, err1 = time.ParseInLocation("2006-01-02T15:04:05", event.StartDatetime, time.Local)
	if err1 != nil {
		logger.Error(fmt.Sprintf("Could not parse %v into time", event.StartDatetime), err1)
		return entry, err1
	}
	entry.EndTime, err2 = time.ParseInLocation("2006-01-02T15:04:05", event.EndDatetime, time.Local)
	if err2 != nil {
		logger.Error(fmt.Sprintf("Could not parse %v into time", event.EndDatetime), err2)
		return entry, err2
	}
	entry.Duration = entry.EndTime.Sub(entry.StartTime)
	entry.EventId = event.EventID
	entry.Live = event.Live
	return entry, nil
}

// parseDuration is a helper function converting calCms duration into seconds for display purposes
func parseDuration(dur string) string {
	dStr := dur[0:2] + "h" + dur[3:5] + "m" + dur[6:8] + "s"
	d, err := time.ParseDuration(dStr)
	if err != nil {
		return "N/A"
	}
	return strconv.FormatFloat(math.Round(d.Seconds()/60), 'f', 1, 64)
}

// extractFileInfo is a helper function that returns file status and file duration as strings
func extractFileInfo(files *domain.FileList, hashEnabled bool) (fileStatus string, duration string) {
	var (
		filesIdentical bool
		hash           string
	)
	if len(*files) == 1 {
		return "Present", strconv.FormatFloat(math.Round((*files)[0].Duration/60), 'f', 1, 64)
	}
	if hashEnabled {
		filesIdentical = true
		for _, file := range *files {
			if file.Checksum == "" {
				return "Multiple", "N/A"
			} else {
				if hash == "" {
					hash = file.Checksum
				} else {
					filesIdentical = (hash == file.Checksum)
				}
			}
		}
		if filesIdentical {
			return "Multiple (identical)", strconv.FormatFloat(math.Round((*files)[0].Duration/60), 'f', 1, 64)
		} else {
			return "Multiple (different)", "N/A"
		}
	} else {
		return "Multiple", "N/A"
	}

}

// convertEvent is a helper function that converts calCms data into the event representation
func (s DefaultCalCmsService) convertEvent(calCmsData domain.CalCmsPgmData) []dto.Event {
	var (
		el []dto.Event
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
				files := s.Repo.GetByEventId(event.EventID)
				if files == nil {
					ev.FileStatus = "Missing"
					ev.ActualDuration = "N/A"
				} else {
					ev.FileStatus, ev.ActualDuration = extractFileInfo(files, s.Cfg.Crawl.GenerateHash)
				}
			} else {
				ev.EventType = "Live"
				ev.FileStatus = "N/A"
				ev.ActualDuration = "N/A"
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
		data, err := s.getCalCmsData()
		if err != nil {
			logger.Error("error getting data from calCms", err)
			return nil, err
		}
		if err := json.Unmarshal(data, &calCmsData); err != nil {
			logger.Error("Cannot convert calCMS response data to Json", err)
			return nil, err
		}
		el := s.convertEvent(calCmsData)
		return el, nil
	} else {
		logger.Warn("calCMS query not enabled in configuration. Not querying.")
		return nil, nil
	}
}
