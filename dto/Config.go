// package dto defines the data structures used to exchange information
package dto

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/johannes-kuhfuss/mairlist-feeder/appstate"
	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/robfig/cron/v3"
)

// ConfigResp converted configuration data for display on the web UI
type ConfigResp struct {
	ServerHost                 string
	ServerPort                 string
	ServerTLSPort              string
	ServerGracefulShutdownTime string
	ServerUseTLS               string
	ServerCertFile             string
	ServerKeyFile              string
	GinMode                    string
	StartDate                  string
	RootFolder                 string
	FileExtensions             string
	AudioFileExtensions        string
	StreamFileExtensions       string
	StreamFileMapping          string
	CycleTime                  string
	ExportFolder               string
	AppendToPlayout            string
	ShortAllowance             string
	LongAllowance              string
	CrawlRunNumber             string
	LastCrawlDate              string
	LastExportDate             string
	LastExportedFileDate       string
	LastExportFileName         string
	CrawlRunning               string
	ExportRunning              string
	CleanRunning               string
	LastCleanDate              string
	NextCrawlDate              string
	NextExportDate             string
	NextCleanDate              string
	FilesCleaned               string
	GenHashes                  string
	LastCalCmsState            string
	LastCalCmsRefreshDate      string
	LastCalCmsRefreshError     string
	LastMairListCommState      string
	ExportDayEvents            string
	MairListPlayingState       string
	LogFile                    string
	QueryMairListStatus        string
	ExportLiveItems            string
	AddNonCalCmsFiles          string
	ExportMinute               string
}

// setStartDate sets the service start date and adds the run duration
func setStartDate(date time.Time) string {
	dur := time.Since(date)
	return convertDate(date) + " (running for " + dur.String() + ")"
}

// convertDate converts a date to its display format
func convertDate(date time.Time) string {
	return convertDateInLocation(date, time.Local)
}

func convertDateInLocation(date time.Time, location *time.Location) string {
	if date.IsZero() {
		return "N/A"
	}
	return date.In(location).Format("2006-01-02 15:04:05 -0700 MST")
}

// getNextJobDate retrieves a job's next execution date and returns it in its display format
func getNextJobDate(bgJobs *cron.Cron, jobID cron.EntryID) string {
	if bgJobs != nil && bgJobs.Entry(jobID).Valid() {
		return bgJobs.Entry(jobID).Next.String()
	} else {
		return "N/A"
	}
}

// getStreamMappings retrieves the mapping between the stream names and stream IDs and returns it in its display format
func getStreamMappings(mappings map[string]int) (mapStr string) {
	keys := make([]string, 0, len(mappings))
	for k := range mappings {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		mapStr = mapStr + k + " -> " + strconv.Itoa(mappings[k]) + "; "
	}
	return
}

func formatLogFile(logFile string) string {
	if logFile == "" {
		return "Logging to file disabled"
	}
	return logFile
}

// GetConfig converts the configuration to its display format
func GetConfig(cfg *config.AppConfig, state *appstate.AppState) (resp ConfigResp) {
	runtime := state.Runtime.Snapshot()
	resp = ConfigResp{
		ServerHost:                 cfg.Server.Host,
		ServerPort:                 cfg.Server.Port,
		ServerTLSPort:              cfg.Server.TLSPort,
		ServerGracefulShutdownTime: strconv.Itoa(cfg.Server.GracefulShutdownTime),
		ServerUseTLS:               strconv.FormatBool(cfg.Server.UseTLS),
		ServerCertFile:             cfg.Server.CertFile,
		ServerKeyFile:              cfg.Server.KeyFile,
		GinMode:                    cfg.Gin.Mode,
		RootFolder:                 cfg.Crawl.RootFolder,
		FileExtensions:             strings.Join(cfg.Crawl.CrawlExtensions, ", "),
		AudioFileExtensions:        strings.Join(cfg.Crawl.AudioFileExtensions, ", "),
		StreamFileExtensions:       strings.Join(cfg.Crawl.StreamingFileExtensions, ", "),
		CycleTime:                  strconv.Itoa(cfg.Crawl.CrawlCycleMin),
		ExportFolder:               cfg.Export.ExportFolder,
		AppendToPlayout:            strconv.FormatBool(cfg.Export.AppendPlaylist),
		ShortAllowance:             strconv.FormatFloat(cfg.Export.ShortDeltaAllowance, 'f', 1, 64),
		LongAllowance:              strconv.FormatFloat(cfg.Export.LongDeltaAllowance, 'f', 1, 64),
		CrawlRunNumber:             strconv.Itoa(runtime.CrawlRunNumber),
		CrawlRunning:               strconv.FormatBool(runtime.CrawlRunning),
		ExportRunning:              strconv.FormatBool(runtime.ExportRunning),
		CleanRunning:               strconv.FormatBool(runtime.CleanRunning),
		FilesCleaned:               strconv.Itoa(runtime.FilesCleaned),
		GenHashes:                  strconv.FormatBool(cfg.Crawl.GenerateHash),
		LastCalCmsState:            runtime.LastCalCmsState,
		LastCalCmsRefreshDate:      convertDate(runtime.LastCalCmsRefreshDate),
		LastCalCmsRefreshError:     formatEmpty(runtime.LastCalCmsRefreshErr),
		LastMairListCommState:      runtime.LastMairListCommState,
		ExportDayEvents:            strconv.FormatBool(cfg.CalCms.ExportDayEvents),
		MairListPlayingState:       strconv.FormatBool(runtime.MairListPlaying),
		LogFile:                    formatLogFile(cfg.Server.LogFile),
		QueryMairListStatus:        strconv.FormatBool(cfg.Export.QueryMairListStatus),
		ExportLiveItems:            strconv.FormatBool(cfg.Export.ExportLiveItems),
		AddNonCalCmsFiles:          strconv.FormatBool(cfg.Crawl.AddNonCalCmsFiles),
		ExportMinute:               strconv.Itoa(cfg.Export.ExportMinute),
	}
	resp.LastCrawlDate = convertDate(runtime.LastCrawlDate)
	resp.LastExportDate = convertDate(runtime.LastExportRunDate)
	resp.LastCleanDate = convertDate(runtime.LastCleanDate)
	resp.LastExportedFileDate = convertDate(runtime.LastExportedFileDate)
	resp.StartDate = setStartDate(runtime.StartDate)
	if runtime.LastExportFileName == "" {
		resp.LastExportFileName = "N/A"
	} else {
		resp.LastExportFileName = runtime.LastExportFileName
	}
	if cfg.Server.Host == "" {
		resp.ServerHost = "localhost"
	}
	resp.NextCrawlDate = getNextJobDate(runtime.BgJobs, runtime.CrawlJobID)
	resp.NextCleanDate = getNextJobDate(runtime.BgJobs, runtime.CleanJobID)
	resp.NextExportDate = getNextJobDate(runtime.BgJobs, runtime.ExportJobID)
	resp.StreamFileMapping = getStreamMappings(cfg.Crawl.StreamMap)
	return
}

func formatEmpty(value string) string {
	if value == "" {
		return "N/A"
	}
	return value
}
