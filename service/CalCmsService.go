package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
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
	if s.Cfg.CalCms.QueryCalCms {
		calUrl, err := url.Parse(s.Cfg.CalCms.CmsUrl)
		if err != nil {
			logger.Error("Cannot parse CalCms Url", err)
			return
		}
		query := url.Values{}
		query.Add("date", time.Now().Format("2006-01-02"))
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
		s.EnrichFileInformation()
		return
	} else {
		logger.Warn("CalCms query not enabled in configuration. Not querying.")
		return
	}
}

func (s DefaultCalCmsService) EnrichFileInformation() {
	var (
		newFile domain.FileInfo
	)
	logger.Info("Starting to add information from calCMS...")
	files := s.Repo.GetAll()
	if len(*files) > 0 {
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
				newFile.EndTime = info.EndTime.Format("15:04")
				newFile.CalCmsTitle = info.Title
				newFile.CalCmsInfoExtracted = true
				err = s.Repo.Store(newFile)
				if err != nil {
					logger.Error("Error updating information in file repository", err)
				}
			}
		}
	}
	logger.Info("Done adding information from calCMS.")
}

func (s DefaultCalCmsService) checkCalCmsData(file domain.FileInfo) (*dto.CalCmsEntry, error) {
	info, err := s.GetCalCmsDataForId(file.EventId)
	if err != nil {
		logger.Error("Error retrieving calCMS info: ", err)
		return nil, err
	}
	if len(info) == 0 {
		logger.Warn(fmt.Sprintf("No information from calCMS for Id %v", file.EventId))
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
	if file.StartTime != info[0].StartTime.Format("15:04") {
		logger.Warn(fmt.Sprintf("Start times differ. File: %v, calCMS: %v. Not adding information.", file.StartTime, info[0].StartTime.Format("15:04")))
		return nil, errors.New("start time difference between file and calCMS")
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
			if event.EventID == strconv.Itoa(id) {
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
	entry.StartTime, err1 = time.Parse("15:04", event.StartTimeName)
	if err1 != nil {
		logger.Error(fmt.Sprintf("Could not parse %v into time", event.StartTimeName), err1)
		return entry, err1
	}
	entry.EndTime, err2 = time.Parse("15:04", event.EndTimeName)
	if err2 != nil {
		logger.Error(fmt.Sprintf("Could not parse %v into time", event.EndTimeName), err2)
		return entry, err2
	}
	if (err1 == nil) && (err2 == nil) {
		entry.Duration = entry.EndTime.Sub(entry.StartTime)
	}
	id, err := strconv.Atoi(event.EventID)
	if err != nil {
		logger.Error(fmt.Sprintf("Could not parse %v into time", event.EndTimeName), err)
		return entry, err
	}
	entry.EventId = id
	entry.Live = event.Live
	return entry, nil
}
