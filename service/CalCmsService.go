package service

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/domain"
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
	calCmsPgm     domain.CalCmsPgmData
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

func (s DefaultCalCmsService) Query() error {
	if s.Cfg.CalCms.QueryCalCms {
		calUrl, err := url.Parse(s.Cfg.CalCms.CmsUrl)
		if err != nil {
			logger.Error("Cannot parse CalCms Url", err)
			return err
		}
		query := url.Values{}
		query.Add("date", time.Now().Format("2006-01-02"))
		query.Add("template", s.Cfg.CalCms.Template)
		calUrl.RawQuery = query.Encode()
		req, err := http.NewRequest("GET", calUrl.String(), nil)
		if err != nil {
			logger.Error("Cannot build CalCms http request", err)
			return err
		}
		resp, err := httpCalClient.Do(req)
		if err != nil {
			logger.Error("Cannot execute CalCms http request", err)
			return err
		}
		defer resp.Body.Close()
		bData, err := io.ReadAll(resp.Body)
		if err != nil {
			logger.Error("Cannot read response data from CalCms http request", err)
			return err
		}
		err = json.Unmarshal(bData, &calCmsPgm)
		if err != nil {
			logger.Error("Cannot convert CalCms response data to Json", err)
			return err
		}
		return nil
	} else {
		logger.Warn("CalCms query not enabled in configuration. Not querying.")
		return nil
	}
}
