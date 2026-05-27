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
	ServerTlsPort              string
	ServerGracefulShutdownTime string
	ServerUseTls               string
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
	if date.IsZero() {
		return "N/A"
	} else {
		return date.Local().Format("2006-01-02 15:04:05 -0700 MST")
	}
}

// getNextJobDate retrieves a job's next execution date and returns it in its display format
func getNextJobDate(state *appstate.AppState, jobId cron.EntryID) string {
	if state.Runtime.BgJobs != nil && state.Runtime.BgJobs.Entry(jobId).Valid() {
		return state.Runtime.BgJobs.Entry(jobId).Next.String()
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
	state.Runtime.Mu.Lock()
	defer state.Runtime.Mu.Unlock()
	resp = ConfigResp{
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
		AudioFileExtensions:        strings.Join(cfg.Crawl.AudioFileExtensions, ", "),
		StreamFileExtensions:       strings.Join(cfg.Crawl.StreamingFileExtensions, ", "),
		CycleTime:                  strconv.Itoa(cfg.Crawl.CrawlCycleMin),
		ExportFolder:               cfg.Export.ExportFolder,
		AppendToPlayout:            strconv.FormatBool(cfg.Export.AppendPlaylist),
		ShortAllowance:             strconv.FormatFloat(cfg.Export.ShortDeltaAllowance, 'f', 1, 64),
		LongAllowance:              strconv.FormatFloat(cfg.Export.LongDeltaAllowance, 'f', 1, 64),
		CrawlRunNumber:             strconv.Itoa(state.Runtime.CrawlRunNumber),
		CrawlRunning:               strconv.FormatBool(state.Runtime.CrawlRunning),
		ExportRunning:              strconv.FormatBool(state.Runtime.ExportRunning),
		CleanRunning:               strconv.FormatBool(state.Runtime.CleanRunning),
		FilesCleaned:               strconv.Itoa(state.Runtime.FilesCleaned),
		GenHashes:                  strconv.FormatBool(cfg.Crawl.GenerateHash),
		LastCalCmsState:            state.Runtime.LastCalCmsState,
		LastCalCmsRefreshDate:      convertDate(state.Runtime.LastCalCmsRefreshDate),
		LastCalCmsRefreshError:     formatEmpty(state.Runtime.LastCalCmsRefreshErr),
		LastMairListCommState:      state.Runtime.LastMairListCommState,
		ExportDayEvents:            strconv.FormatBool(cfg.CalCms.ExportDayEvents),
		MairListPlayingState:       strconv.FormatBool(state.Runtime.MairListPlaying),
		LogFile:                    formatLogFile(cfg.Server.LogFile),
		QueryMairListStatus:        strconv.FormatBool(cfg.Export.QueryMairListStatus),
		ExportLiveItems:            strconv.FormatBool(cfg.Export.ExportLiveItems),
		AddNonCalCmsFiles:          strconv.FormatBool(cfg.Crawl.AddNonCalCmsFiles),
		ExportMinute:               strconv.Itoa(cfg.Export.ExportMinute),
	}
	resp.LastCrawlDate = convertDate(state.Runtime.LastCrawlDate)
	resp.LastExportDate = convertDate(state.Runtime.LastExportRunDate)
	resp.LastCleanDate = convertDate(state.Runtime.LastCleanDate)
	resp.LastExportedFileDate = convertDate(state.Runtime.LastExportedFileDate)
	resp.StartDate = setStartDate(state.Runtime.StartDate)
	if state.Runtime.LastExportFileName == "" {
		resp.LastExportFileName = "N/A"
	} else {
		resp.LastExportFileName = state.Runtime.LastExportFileName
	}
	if cfg.Server.Host == "" {
		resp.ServerHost = "localhost"
	}
	resp.NextCrawlDate = getNextJobDate(state, state.Runtime.CrawlJobId)
	resp.NextCleanDate = getNextJobDate(state, state.Runtime.CleanJobId)
	resp.NextExportDate = getNextJobDate(state, state.Runtime.ExportJobId)
	resp.StreamFileMapping = getStreamMappings(cfg.Crawl.StreamMap)
	return
}

func formatEmpty(value string) string {
	if value == "" {
		return "N/A"
	}
	return value
}
