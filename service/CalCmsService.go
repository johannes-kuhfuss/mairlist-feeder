package service

import (
	"encoding/json"
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
	calCmsPgm     struct {
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
		calCmsPgm.Lock()
		err = json.Unmarshal(bData, &calCmsPgm.data)
		calCmsPgm.Unlock()
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
	// Get all entries from file list
	// for every entry that has an Id
	// extract information entry from calcms
	// compare start time, error if different
	// check fromcalcms = true, warn if not set
	// add end time
}

func (s DefaultCalCmsService) GetCalCmsDataForHour(hour string) ([]dto.CalCmsEntry, error) {
	var entries []dto.CalCmsEntry
	calCmsPgm.RLock()
	events := calCmsPgm.data.Events
	calCmsPgm.RUnlock()
	if len(events) > 0 {
		for _, event := range events {
			if (event.Live == 0) && (strings.HasPrefix(event.StartTimeName, hour)) {
				entry, err := s.convertToEntry(event)
				if err != nil {
					entries = append(entries, entry)
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
	return entry, nil
}
