// package service implements the services and their business logic that provide the main part of the program
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
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

type CalCmsQuerier interface {
	Query() error
	QueryContext(context.Context) error
}

// The calCms service handles all the communication with calCms and the necessary data transformation
type DefaultCalCmsService struct {
	Cfg             *config.AppConfig
	State           *appstate.AppState
	Repo            *repositories.DefaultFileRepository
	httpClient      *http.Client
	Now             func() time.Time
	calCmsPgm       *safeCalCmsPgm
	eventsToday     *safeEvents
	eventsYesterday *safeEvents
}

type safeCalCmsPgm struct {
	sync.RWMutex
	data domain.CalCmsPgmData
}

type safeEvents struct {
	sync.RWMutex
	events []dto.Event
}

// InitHttpCalClient sets the defaukt values for the http client used to query calCms
func InitHttpCalClient() *http.Client {
	httpCalTr := http.Transport{
		DisableKeepAlives:  false,
		DisableCompression: false,
		MaxIdleConns:       0,
		IdleConnTimeout:    0,
	}
	return &http.Client{
		Transport: &httpCalTr,
		Timeout:   5 * time.Second,
	}
}

// NewCalCmsService creates a new calCms service and injects its dependencies
func NewCalCmsService(cfg *config.AppConfig, repo *repositories.DefaultFileRepository) DefaultCalCmsService {
	return NewCalCmsServiceWithState(cfg, appstate.New(), repo)
}

func NewCalCmsServiceWithState(cfg *config.AppConfig, state *appstate.AppState, repo *repositories.DefaultFileRepository) DefaultCalCmsService {
	return DefaultCalCmsService{
		Cfg:             cfg,
		State:           state,
		Repo:            repo,
		httpClient:      InitHttpCalClient(),
		Now:             time.Now,
		calCmsPgm:       &safeCalCmsPgm{},
		eventsToday:     &safeEvents{},
		eventsYesterday: &safeEvents{},
	}
}

// insertData inserts new calCms data in a thread-safe manner
func (s DefaultCalCmsService) insertData(data domain.CalCmsPgmData) {
	s.calCmsPgm.Lock()
	defer s.calCmsPgm.Unlock()
	s.calCmsPgm.data = data
}

// calcCalCmsEndDate calculates the next-day end date based on a given start date.
func calcCalCmsEndDate(startDate string) (endDate string, e error) {
	return calcCalCmsEndDateForDays(startDate, 1)
}

// calcCalCmsEndDateForDays calculates the exclusive end date for a multi-day calCMS query.
func calcCalCmsEndDateForDays(startDate string, days int) (endDate string, e error) {
	sd, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return "", err
	}
	return sd.AddDate(0, 0, days).Format("2006-01-02"), nil
}

// setCalCmsQueryState sets staus of last calCms interaction with result and time for status overview
func (s DefaultCalCmsService) setCalCmsQueryState(success bool) {
	s.State.Runtime.Update(func(runtime *appstate.RuntimeState) {
		if success {
			runtime.LastCalCmsState = fmt.Sprintf("Succeeded (%v)", s.Now().Format("2006-01-02 15:04:05 -0700 MST"))
		} else {
			runtime.LastCalCmsState = fmt.Sprintf("Failed (%v)", s.Now().Format("2006-01-02 15:04:05 -0700 MST"))
		}
	})
	if success {
		s.State.Metrics.SetConnected("calCMS", 1)
	} else {
		s.State.Metrics.SetConnected("calCMS", 0)
	}

}

// getCalCmsEventData retrieves the event information for the crawled date range from calCms.
func (s DefaultCalCmsService) getCalCmsEventData() (eventData []byte, e error) {
	return s.getCalCmsEventDataContext(context.Background())
}

func (s DefaultCalCmsService) getCalCmsEventDataContext(ctx context.Context) (eventData []byte, e error) {
	return s.getCalCmsEventDataForDatesContext(ctx, helper.GetCrawlDates(s.Cfg.Misc.TestCrawl, s.Cfg.Misc.TestDate))
}

func (s DefaultCalCmsService) getCalCmsEventDataForDates(dates []time.Time) (eventData []byte, e error) {
	return s.getCalCmsEventDataForDatesContext(context.Background(), dates)
}

func (s DefaultCalCmsService) getCalCmsEventDataForDatesContext(ctx context.Context, dates []time.Time) (eventData []byte, e error) {
	//API doc: https://github.com/rapilodev/racalmas/blob/master/docs/event-api.md
	//URL old: https://programm.coloradio.org/agenda/events.cgi?date=2024-04-09&template=event.json-p
	//URL new: https://programm.coloradio.org/agenda/events.cgi?from_date=2024-10-04&from_time=00:00&till_date=2024-10-05&till_time=00:00&template=event.json-p
	if len(dates) == 0 {
		return nil, errors.New("no calCMS query dates configured")
	}
	calUrl, err := url.Parse(s.Cfg.CalCms.CmsUrl)
	if err != nil {
		logger.Error("Cannot parse calCMS Url", err)
		return nil, err
	}
	query := url.Values{}
	calCmsStartDate := domain.FormatFolderDate(dates[0])
	calCmsEndDate, err := calcCalCmsEndDateForDays(calCmsStartDate, len(dates))
	if err != nil {
		return nil, err
	}
	query.Add("from_date", calCmsStartDate)
	query.Add("from_time", "00:00")
	query.Add("till_date", calCmsEndDate)
	query.Add("till_time", "00:00")
	query.Add("template", s.Cfg.CalCms.Template)
	calUrl.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, calUrl.String(), nil)
	if err != nil {
		s.setCalCmsQueryState(false)
		logger.Error("Cannot build calCMS http request", err)
		return nil, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.setCalCmsQueryState(false)
		logger.Error("Cannot execute calCMS http request", err)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		s.setCalCmsQueryState(false)
		err := errors.New(resp.Status)
		logger.Errorf("Received status code %v from calCMS. %v", resp.StatusCode, err)
		return nil, err
	}
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
	return s.QueryContext(context.Background())
}

func (s DefaultCalCmsService) QueryContext(ctx context.Context) error {
	if s.Cfg.CalCms.QueryCalCms {
		logger.Info("Starting to add information from calCMS...")
		start := s.Now().UTC()
		data, err := s.getCalCmsEventDataContext(ctx)
		if err != nil {
			logger.Error("error getting data from calCms", err)
			return err
		}
		var calCmsData domain.CalCmsPgmData
		if err := json.Unmarshal(data, &calCmsData); err != nil {
			logger.Error("Cannot convert calCMS response data to Json", err)
			return err
		}
		s.insertData(calCmsData)
		fc := s.EnrichFileInformation()
		end := s.Now().UTC()
		updateDur := end.Sub(start)
		s.State.Metrics.ObserveFastEvent("lastcalcmsupdate", updateDur.Seconds())
		logger.Infof("Added or updated information from calCMS for %v file(s), audio: %v, stream: %v (%v)", fc.TotalCount, fc.AudioCount, fc.StreamCount, updateDur.String())
		return nil
	}
	logger.Warn("calCMS query not enabled in configuration. Not querying")
	return nil

}

// EnrichFileInformation runs through all file representations and adds information from calCms where applicable
func (s DefaultCalCmsService) EnrichFileInformation() (fc dto.FileCounts) {
	for _, folderDate := range helper.GetCrawlDates(s.Cfg.Misc.TestCrawl, s.Cfg.Misc.TestDate) {
		if files := s.Repo.GetByDate(folderDate); files != nil {
			for _, file := range files {
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
	if oldFileInfo.FileType == domain.FileTypeAudio {
		fc.AudioCount++
	}
	if (oldFileInfo.FileType == domain.FileTypeStream) && (oldFileInfo.StreamId != 0) {
		newFileInfo.Duration = calCmsInfo.Duration
		fc.StreamCount++
	}
	newFileInfo.EventIsLive = calCmsInfo.Live
	fc.TotalCount++
	return
}

// checkCalCmsEventData evaluates calCms event data on a per file basis and performs some sanity checks
func (s DefaultCalCmsService) checkCalCmsEventData(file domain.FileInfo) (*dto.CalCmsEntry, error) {
	var (
		exportLive string
	)
	allInfo, err := s.GetCalCmsEventDataForId(file.EventId)
	if err != nil {
		return nil, err
	}
	var info []dto.CalCmsEntry
	normalizedFileDate := domain.NormalizeDate(file.FolderDate)
	for _, entry := range allInfo {
		if domain.NormalizeDate(entry.StartTime).Equal(normalizedFileDate) {
			info = append(info, entry)
		}
	}
	if len(info) == 0 {
		if len(allInfo) > 0 {
			return nil, fmt.Errorf("file has different date (%v) than calCms data (%v)", domain.FormatFolderDate(file.FolderDate), domain.FormatFolderDate(allInfo[0].StartTime))
		}
		return nil, fmt.Errorf("no Id %v in calCMS", file.EventId)
	}
	if len(info) > 1 {
		logger.Warnf("Ambiguous information from calCMS. Found %v entries. Not adding information.", len(info))
		return nil, errors.New("multiple matches in calCMS")
	}
	calCmsDate := domain.NormalizeDate(info[0].StartTime)
	if !calCmsDate.Equal(domain.NormalizeDate(file.FolderDate)) {
		return nil, fmt.Errorf("file has different date (%v) than calCms data (%v)", domain.FormatFolderDate(file.FolderDate), domain.FormatFolderDate(calCmsDate))
	}
	if (len(info) == 1) && info[0].Live {
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
	s.calCmsPgm.RLock()
	events := append([]domain.CalCmsEvent(nil), s.calCmsPgm.data.Events...)
	s.calCmsPgm.RUnlock()
	if len(events) > 0 {
		for _, event := range events {
			if (event.Live == 0) && (strings.HasPrefix(event.StartTimeName, hour)) {
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

// GetCalCmsEventDataForId retrieves all event data from the calCms data for a given Event Id
func (s DefaultCalCmsService) GetCalCmsEventDataForId(id int) (entries []dto.CalCmsEntry, e error) {
	s.calCmsPgm.RLock()
	events := append([]domain.CalCmsEvent(nil), s.calCmsPgm.data.Events...)
	s.calCmsPgm.RUnlock()
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

// GetCalCmsEventDataForIdAndDate retrieves all event data from the calCms data for a given Event Id and date.
func (s DefaultCalCmsService) GetCalCmsEventDataForIdAndDate(id int, folderDate time.Time) (entries []dto.CalCmsEntry, e error) {
	allEntries, err := s.GetCalCmsEventDataForId(id)
	if err != nil {
		return nil, err
	}
	normalizedDate := domain.NormalizeDate(folderDate)
	for _, entry := range allEntries {
		if domain.NormalizeDate(entry.StartTime).Equal(normalizedDate) {
			entries = append(entries, entry)
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
	entry.Live = event.Live != 0
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
func extractFileInfo(files domain.FileList, hashEnabled bool) (fileStatus string, duration string, fileSource string) {
	var fs string
	if len(files) == 0 {
		return "N/A", "N/A", "N/A"
	}
	if len(files) == 1 {
		if files[0].FromCalCMS && files[0].EventId != 0 {
			fs = "calCMS"
		} else {
			fs = "Manual"
		}
		return "Present", strconv.FormatFloat(math.Round(files[0].Duration.Minutes()), 'f', 1, 64), fs
	}
	if hashEnabled {
		filesIdentical, checksumAvail := checkHash(files)
		switch {
		case checksumAvail && filesIdentical:
			return "Multiple (identical)", strconv.FormatFloat(math.Round(files[0].Duration.Minutes()), 'f', 1, 64), "N/A"
		case checksumAvail && !filesIdentical:
			return "Multiple (different)", "N/A", "N/A"
		default:
			return "Multiple", "N/A", "N/A"
		}
	}
	return "Multiple", "N/A", "N/A"
}

// checkHash compares the has of all files and returns true, if the hash values of all files are identical
func checkHash(files domain.FileList) (filesIdentical bool, checksumAvail bool) {
	var (
		hash string
	)
	if len(files) < 2 {
		return false, false
	}
	filesIdentical = true
	for _, file := range files {
		if file.Checksum == "" {
			return false, false
		} else {
			if hash == "" {
				hash = file.Checksum
				continue
			}
			if hash != file.Checksum {
				return false, true
			}
		}
	}
	return filesIdentical, true
}

// convertEvent is a helper function that converts calCms data into the event representation
func (s DefaultCalCmsService) convertEvent(calCmsData domain.CalCmsPgmData) []dto.Event {
	var (
		el    []dto.Event
		files domain.FileList
	)
	for _, event := range calCmsData.Events {
		if !slices.Contains(s.Cfg.CalCms.EventExclusion, event.Skey) {
			var ev dto.Event
			ev.CurrentEvent = isCurrent(event.StartDate, event.StartTime, event.EndTime)
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
				hour, err := hourFromTimeString(ev.StartTime)
				if err != nil {
					logger.Errorf("Could not extract event hour from %q. %v", ev.StartTime, err)
				} else {
					eventDate, err := domain.ParseFolderDate(ev.StartDate)
					if err != nil {
						logger.Errorf("Could not extract event date from %q. %v", ev.StartDate, err)
					} else {
						files = s.Repo.GetByIdAndDateAndHour(event.EventID, eventDate, hour, s.Cfg.Export.ExportLiveItems)
					}
				}
			} else {
				eventDate, err := domain.ParseFolderDate(ev.StartDate)
				if err != nil {
					logger.Errorf("Could not extract event date from %q. %v", ev.StartDate, err)
				} else {
					files = s.Repo.GetByEventIdAndDate(event.EventID, eventDate)
				}
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

func isCurrent(eventDate, startTime, endTime string) string {
	return isCurrentAt(eventDate, startTime, endTime, time.Now())
}

func isCurrentAt(eventDate, startTime, endTime string, now time.Time) string {
	date, err := domain.ParseFolderDate(eventDate)
	if err != nil {
		logger.Errorf("Could not parse event date %q. %v", eventDate, err)
		return ""
	}
	sth, stm, err := parseCompactHourMinute(startTime)
	if err != nil {
		logger.Errorf("Could not parse event start time %q. %v", startTime, err)
		return ""
	}
	eth, etm, err := parseCompactHourMinute(endTime)
	if err != nil {
		logger.Errorf("Could not parse event end time %q. %v", endTime, err)
		return ""
	}
	st := time.Date(date.Year(), date.Month(), date.Day(), sth, stm, 0, 0, time.Local)
	et := time.Date(date.Year(), date.Month(), date.Day(), eth, etm, 0, 0, time.Local)
	if !et.After(st) {
		et = et.AddDate(0, 0, 1)
	}
	if now.After(st) && now.Before(et) {
		return "***"
	}
	return ""
}

func parseCompactHourMinute(timeValue string) (hour int, minute int, err error) {
	compact := strings.ReplaceAll(timeValue, ":", "")
	if len(compact) < 4 {
		return 0, 0, fmt.Errorf("time value is shorter than HHMM")
	}
	hour, err = strconv.Atoi(compact[0:2])
	if err != nil {
		return 0, 0, err
	}
	minute, err = strconv.Atoi(compact[2:4])
	if err != nil {
		return 0, 0, err
	}
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, 0, fmt.Errorf("time value is outside valid range")
	}
	return hour, minute, nil
}

func hourFromTimeString(timeValue string) (string, error) {
	hour, _, err := parseCompactHourMinute(timeValue)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%02d", hour), nil
}

// RefreshTodayEvents updates the cached event list for display on the web UI.
func (s DefaultCalCmsService) RefreshTodayEvents() ([]dto.Event, error) {
	return s.RefreshTodayEventsContext(context.Background())
}

func (s DefaultCalCmsService) RefreshTodayEventsContext(ctx context.Context) ([]dto.Event, error) {
	var (
		calCmsData domain.CalCmsPgmData
	)
	if s.Cfg.CalCms.QueryCalCms {
		data, err := s.getCalCmsEventDataContext(ctx)
		if err != nil {
			logger.Error("error getting data from calCms", err)
			s.setTodayRefreshState(err)
			return nil, err
		}
		if err := json.Unmarshal(data, &calCmsData); err != nil {
			logger.Error("Cannot convert calCMS response data to Json", err)
			s.setTodayRefreshState(err)
			return nil, err
		}
		s.insertData(calCmsData)
		el := s.convertEvent(calCmsData)
		s.eventsToday.Lock()
		s.eventsToday.events = append([]dto.Event(nil), el...)
		s.eventsToday.Unlock()
		s.countEvents(el)
		s.setTodayRefreshState(nil)
		return el, nil
	} else {
		logger.Warn("calCMS query not enabled in configuration. Not querying.")
		return nil, nil
	}
}

func (s DefaultCalCmsService) setTodayRefreshState(err error) {
	s.State.Runtime.Update(func(runtime *appstate.RuntimeState) {
		runtime.LastCalCmsRefreshDate = s.Now()
		if err != nil {
			runtime.LastCalCmsRefreshErr = err.Error()
			return
		}
		runtime.LastCalCmsRefreshErr = ""
	})
}

// GetTodayEvents returns the cached event list for display on the web UI.
func (s DefaultCalCmsService) GetTodayEvents() ([]dto.Event, error) {
	s.eventsToday.RLock()
	defer s.eventsToday.RUnlock()
	if len(s.eventsToday.events) > 0 {
		return append([]dto.Event(nil), s.eventsToday.events...), nil
	}
	return nil, nil
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
	s.State.Metrics.SetEventCounter("present", float64(presentCount))
	s.State.Metrics.SetEventCounter("missing", float64(missingCount))
	s.State.Metrics.SetEventCounter("multiple", float64(multipleCount))
	s.State.Metrics.SetEventCounter("total", float64(presentCount+missingCount+multipleCount))

}

func (s DefaultCalCmsService) CountRun() {
	s.CountRunContext(context.Background())
}

func (s DefaultCalCmsService) CountRunContext(ctx context.Context) {
	if _, err := s.RefreshTodayEventsContext(ctx); err != nil {
		logger.Error("error refreshing today's events", err)
	}
}

// SaveYesterdaysEvents saves the current event state to the local variable
func (s DefaultCalCmsService) SaveYesterdaysEvents() {
	events, err := s.GetTodayEvents()
	if err == nil {
		snapshotDate := domain.FormatFolderDate(helper.DateForFolder(s.Cfg.Misc.TestCrawl, s.Cfg.Misc.TestDate, 0))
		events = filterEventsForDate(events, snapshotDate)
		s.eventsYesterday.Lock()
		defer s.eventsYesterday.Unlock()
		s.eventsYesterday.events = append([]dto.Event(nil), events...)
	}
}

func filterEventsForDate(events []dto.Event, date string) []dto.Event {
	filteredEvents := make([]dto.Event, 0, len(events))
	for _, event := range events {
		if event.StartDate == date {
			filteredEvents = append(filteredEvents, event)
		}
	}
	return filteredEvents
}

// GetYesterdaysEvents retrieves yesterday's event state from the local variable
func (s DefaultCalCmsService) GetYesterdaysEvents() []dto.Event {
	s.eventsYesterday.RLock()
	defer s.eventsYesterday.RUnlock()
	if len(s.eventsYesterday.events) > 0 {
		return append([]dto.Event(nil), s.eventsYesterday.events...)
	}
	return nil
}
