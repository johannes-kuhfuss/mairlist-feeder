package dto

import (
	"strconv"
	"strings"

	"github.com/johannes-kuhfuss/mairlist-feeder/config"
)

type ConfigResp struct {
	ServerHost                 string
	ServerPort                 string
	ServerTlsPort              string
	ServerGracefulShutdownTime string
	ServerUseTls               string
	ServerCertFile             string
	ServerKeyFile              string
	GinMode                    string
	StartDate                  string
	RootFolder                 string
	FileExtensions             string
	CycleTime                  string
	ExportFolder               string
	ShortAllowance             string
	LongAllowance              string
	CrawlRunNumber             string
	FilesInList                string
}

func GetConfig(cfg *config.AppConfig) ConfigResp {
	resp := ConfigResp{
		ServerHost:                 cfg.Server.Host,
		ServerPort:                 cfg.Server.Port,
		ServerTlsPort:              cfg.Server.TlsPort,
		ServerGracefulShutdownTime: strconv.Itoa(cfg.Server.GracefulShutdownTime),
		ServerUseTls:               strconv.FormatBool(cfg.Server.UseTls),
		ServerCertFile:             cfg.Server.CertFile,
		ServerKeyFile:              cfg.Server.KeyFile,
		GinMode:                    cfg.Gin.Mode,
		StartDate:                  cfg.RunTime.StartDate.Local().Format("2006-01-02 15:04:05 -0700"),
		RootFolder:                 cfg.Crawl.RootFolder,
		FileExtensions:             strings.Join(cfg.Crawl.Extensions, ","),
		CycleTime:                  strconv.Itoa(cfg.Crawl.CrawlCycleMin),
		ExportFolder:               cfg.Export.ExportFolder,
		ShortAllowance:             strconv.FormatFloat(cfg.Export.ShortDeltaAllowance, 'f', 1, 64),
		LongAllowance:              strconv.FormatFloat(cfg.Export.LongDeltaAllowance, 'f', 1, 64),
		CrawlRunNumber:             strconv.Itoa(cfg.RunTime.CrawlRunNumber),
	}
	if cfg.Server.Host == "" {
		resp.ServerHost = "localhost"
	}
	return resp
}
