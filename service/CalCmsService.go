package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/domain"
	"github.com/johannes-kuhfuss/mairlist-feeder/dto"
	"github.com/johannes-kuhfuss/mairlist-feeder/repositories"
	"github.com/johannes-kuhfuss/services_utils/logger"
)

type CalCmsService interface {
	Query()
}

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

func NewCalCmsService(cfg *config.AppConfig, repo *repositories.DefaultFileRepository) DefaultCalCmsService {
	InitHttpCalClient()
	return DefaultCalCmsService{
		Cfg:  cfg,
		Repo: repo,
	}
}

func (s DefaultCalCmsService) insertData(data domain.CalCmsPgmData) {
	CalCmsPgm.Lock()
	CalCmsPgm.data = data
	CalCmsPgm.Unlock()
}

func (s DefaultCalCmsService) Query() {
	//URL: https://programm.coloradio.org/agenda/events.cgi?date=2024-04-09&template=event.json-p
	if s.Cfg.CalCms.QueryCalCms {
		calUrl, err := url.Parse(s.Cfg.CalCms.CmsUrl)
		if err != nil {
			logger.Error("Cannot parse CalCms Url", err)
			return
		}
		query := url.Values{}
		if s.Cfg.Misc.TestCrawl {
			date := strings.ReplaceAll(s.Cfg.Misc.TestDate, "/", "-")
			query.Add("date", date)
		} else {
			query.Add("date", time.Now().Format("2006-01-02"))
		}

		query.Add("template", s.Cfg.CalCms.Template)
		calUrl.RawQuery = query.Encode()
		req, err := http.NewRequest("GET", calUrl.String(), nil)
		if err != nil {
			logger.Error("Cannot build CalCms http request", err)
			return
		}
		resp, err := httpCalClient.Do(req)
		if err != nil {
			logger.Error("Cannot execute CalCms http request", err)
			return
		}
		defer resp.Body.Close()
		bData, err := io.ReadAll(resp.Body)
		if err != nil {
			logger.Error("Cannot read response data from CalCms http request", err)
			return
		}
		CalCmsPgm.Lock()
		err = json.Unmarshal(bData, &CalCmsPgm.data)
		CalCmsPgm.Unlock()
		if err != nil {
			logger.Error("Cannot convert CalCms response data to Json", err)
			return
		}
		enriched := s.EnrichFileInformation()
		logger.Info(fmt.Sprintf("Added information from calCMS for %v files", enriched))
		return
	} else {
		logger.Warn("CalCms query not enabled in configuration. Not querying.")
		return
	}
}

func (s DefaultCalCmsService) EnrichFileInformation() int {
	var (
		newFile        domain.FileInfo
		calCmsEnriched int
	)
	logger.Info("Starting to add information from calCMS...")
	calCmsEnriched = 0
	files := s.Repo.GetAll()
	if files != nil {
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
				calCmsEnriched++
				err = s.Repo.Store(newFile)
				if err != nil {
					logger.Error("Error updating information in file repository", err)
				}
			}
		}
	}
	logger.Info("Done adding information from calCMS.")
	return calCmsEnriched
}

func (s DefaultCalCmsService) checkCalCmsData(file domain.FileInfo) (*dto.CalCmsEntry, error) {
	info, err := s.GetCalCmsDataForId(file.EventId)
	if err != nil {
		logger.Error("Error retrieving calCMS info: ", err)
		return nil, err
	}
	if len(info) == 0 {
		logger.Warn(fmt.Sprintf("No information from calCMS for Id %v in today's calCMS events", file.EventId))
		return nil, errors.New("no such id in calCMS")
	}
	if len(info) != 1 {
		logger.Warn(fmt.Sprintf("Ambiguous information from calCMS. Found %v entries. Not adding information.", len(info)))
		return nil, errors.New("multiple matches in calCMS")
	}
	if info[0].Live == 1 {
		logger.Warn(fmt.Sprintf("%v, Id: %v is designated as live. Not adding information.", info[0].Title, info[0].EventId))
		return nil, errors.New("event is live in calCMS")
	}
	return &info[0], nil
}

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
