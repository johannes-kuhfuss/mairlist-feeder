package dto

import (
	"strconv"
	"strings"
	"time"

	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/robfig/cron/v3"
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
	AppendToPlayout            string
	ShortAllowance             string
	LongAllowance              string
	CrawlRunNumber             string
	LastCrawlDate              string
	FilesInList                string
	AudioFilesInList           string
	StreamFilesInList          string
	LastExportDate             string
	LastExportedFileDate       string
	LastExportFileName         string
	CrawlRunning               string
	ExportRunning              string
	CleanRunning               string
	LimitTime                  string
	LastCleanDate              string
	NextCrawlDate              string
	NextExportDate             string
	NextCleanDate              string
	FilesCleaned               string
}

func convertDate(date time.Time) string {
	if date.IsZero() {
		return "N/A"
	} else {
		return date.Local().Format("2006-01-02 15:04:05 -0700 MST")
	}
}

func getNextJobDate(cfg *config.AppConfig, jobId int) string {
	if cfg.RunTime.BgJobs != nil {
		return cfg.RunTime.BgJobs.Entry(cron.EntryID(jobId)).Next.String()
	} else {
		return "N/A"
	}

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
		RootFolder:                 cfg.Crawl.RootFolder,
		FileExtensions:             strings.Join(cfg.Crawl.CrawlExtensions, ", "),
		CycleTime:                  strconv.Itoa(cfg.Crawl.CrawlCycleMin),
		ExportFolder:               cfg.Export.ExportFolder,
		AppendToPlayout:            strconv.FormatBool(cfg.Export.AppendPlaylist),
		ShortAllowance:             strconv.FormatFloat(cfg.Export.ShortDeltaAllowance, 'f', 1, 64),
		LongAllowance:              strconv.FormatFloat(cfg.Export.LongDeltaAllowance, 'f', 1, 64),
		CrawlRunNumber:             strconv.Itoa(cfg.RunTime.CrawlRunNumber),
		FilesInList:                strconv.Itoa(cfg.RunTime.FilesInList),
		AudioFilesInList:           strconv.Itoa(cfg.RunTime.AudioFilesInList),
		StreamFilesInList:          strconv.Itoa(cfg.RunTime.StreamFilesInList),
		CrawlRunning:               strconv.FormatBool(cfg.RunTime.CrawlRunning),
		ExportRunning:              strconv.FormatBool(cfg.RunTime.ExportRunning),
		CleanRunning:               strconv.FormatBool(cfg.RunTime.CleanRunning),
		LimitTime:                  strconv.FormatBool(cfg.Export.LimitTime),
		FilesCleaned:               strconv.Itoa(cfg.RunTime.FilesCleaned),
	}
	resp.LastCrawlDate = convertDate(cfg.RunTime.LastCrawlDate)
	resp.LastExportDate = convertDate(cfg.RunTime.LastExportRunDate)
	resp.LastCleanDate = convertDate(cfg.RunTime.LastCleanDate)
	resp.LastExportedFileDate = convertDate(cfg.RunTime.LastExportedFileDate)
	resp.StartDate = convertDate(cfg.RunTime.StartDate)
	if cfg.RunTime.LastExportFileName == "" {
		resp.LastExportFileName = "N/A"
	} else {
		resp.LastExportFileName = cfg.RunTime.LastExportFileName
	}
	if cfg.Server.Host == "" {
		resp.ServerHost = "localhost"
	}
	resp.NextCrawlDate = getNextJobDate(cfg, cfg.RunTime.CrawlJobId)
	resp.NextCleanDate = getNextJobDate(cfg, cfg.RunTime.CleanJobId)
	resp.NextExportDate = getNextJobDate(cfg, cfg.RunTime.ExportJobId)
	return resp
}
