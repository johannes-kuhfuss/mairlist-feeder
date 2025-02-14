// package config defines the program's configuration including the defaults
package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/robfig/cron/v3"
)

// Configuration with subsections
type AppConfig struct {
	Server struct {
		Host                 string `envconfig:"SERVER_HOST"`
		Port                 string `envconfig:"SERVER_PORT" default:"8080"`
		TlsPort              string `envconfig:"SERVER_TLS_PORT" default:"8443"`
		GracefulShutdownTime int    `envconfig:"GRACEFUL_SHUTDOWN_TIME" default:"10"`
		UseTls               bool   `envconfig:"USE_TLS" default:"false"`
		CertFile             string `envconfig:"CERT_FILE" default:"./cert/cert.pem"`
		KeyFile              string `envconfig:"KEY_FILE" default:"./cert/cert.key"`
		LogFile              string `envconfig:"LOG_FILE"` // leave empty to disable logging to file
	}
	Gin struct {
		Mode         string `envconfig:"GIN_MODE" default:"release"`
		TemplatePath string `envconfig:"TEMPLATE_PATH" default:"./templates/"`
		LogToLogger  bool   `envconfig:"LOG_TO_LOGGER" default:"false"`
	}
	Misc struct {
		TestCrawl    bool   `envconfig:"TEST_CRAWL" default:"false"`
		TestDate     string `envconfig:"TEST_DATE" default:"2024/01/15"`
		FileSaveFile string `envconfig:"FILE_SAVE_FILE" default:"files.dta"`
	}
	Crawl struct {
		RootFolder              string         `envconfig:"ROOT_FOLDER"`
		CrawlExtensions         []string       `envconfig:"CRAWL_EXTENSIONS" default:".mp3,.m4a,.wav,.stream"`
		AudioFileExtensions     []string       `envconfig:"AUDIO_FILE_EXTENSIONS" default:".mp3,.m4a,.wav"`
		StreamingFileExtensions []string       `envconfig:"STREAM_FILE_EXTENSIONS" default:".stream"`
		FfprobePath             string         `envconfig:"FFPROBE_PATH" default:"/usr/bin/ffprobe"`
		FfProbeTimeOut          int            `envconfig:"FFPROBE_TIMEOUT" default:"60"`
		CrawlCycleMin           int            `envconfig:"CRAWL_CYCLE_MIN" default:"15"`
		StreamMap               map[string]int `envconfig:"STREAM_MAP"`
		GenerateHash            bool           `envconfig:"GENERATE_HASH" default:"false"`
		AddNonCalCmsFiles       bool           `envconfig:"ADD_NON_CALCMS_FILES" default:"true"`
	}
	Export struct {
		ExportFolder           string  `envconfig:"EXPORT_FOLDER" default:"C:\\TEMP"`
		ShortDeltaAllowance    float64 `envconfig:"SHORT_DELTA_ALLOWANCE" default:"8.0"`
		LongDeltaAllowance     float64 `envconfig:"LONG_DELTA_ALLOWANCE" default:"12.0"`
		MairListUrl            string  `envconfig:"MAIRLIST_URL" default:"http://localhost:9300/"`
		MairListUser           string  `envconfig:"MAIRLIST_USER" default:"dbtest"`
		MairListPassword       string  `envconfig:"MAIRLIST_PASS" default:"dbtest"`
		MairListVersion        int     `envconfig:"MAIRLIST_VERSION" default:"6"`
		AppendPlaylist         bool    `envconfig:"APPEND_PLAYLIST" default:"false"`
		TerminateAfterDuration bool    `envconfig:"TERM_AFTER_DUR" default:"true"`
		LimitTime              bool    `envconfig:"LIMIT_TIME" default:"false"`
		QueryMairListStatus    bool    `envconfig:"QUERY_MAIRLIST_STATUS" default:"false"`
		StatusQueryCycleSec    int     `envconfig:"QUERY_STATUS_CYCLE_SEC" default:"5"`
		ExportLiveItems        bool    `envconfig:"EXPORT_LIVE_ITEMS" default:"false"`
	}
	CalCms struct {
		QueryCalCms        bool     `envconfig:"QUERY_CALCMS" default:"false"`
		CmsUrl             string   `envconfig:"CALCMS_URL" default:"https://programm.coloradio.org/agenda/events.cgi"`
		Template           string   `envconfig:"CALCMS_TEMPLATE" default:"event.json-p"`
		EventExclusion     []string `envconfig:"EVENT_EXCLUSION"`
		ExportDayEvents    bool     `envconfig:"EXPORT_DAY_EVENTS" default:"false"`
		ShowNonCalCmsFiles bool     `envconfig:"SHOW_NON_CLACMS_FILES" default:"true"`
	}
	Metrics struct {
		FileNumber      prometheus.GaugeVec
		MairListPlaying prometheus.GaugeVec
		Connected       prometheus.GaugeVec
		EventCounters   prometheus.GaugeVec
	}
	RunTime struct {
		Mu                    sync.Mutex
		Router                *gin.Engine
		BgJobs                *cron.Cron
		ListenAddr            string
		StartDate             time.Time
		CrawlRunNumber        int
		LastCrawlDate         time.Time
		FilesInList           int
		AudioFilesInList      int
		StreamFilesInList     int
		LastExportRunDate     time.Time
		LastExportedFileDate  time.Time
		LastExportFileName    string
		CrawlRunning          bool
		ExportRunning         bool
		CleanRunning          bool
		LastCleanDate         time.Time
		FilesCleaned          int
		CrawlJobId            int
		ExportJobId           int
		CleanJobId            int
		EventJobId            int
		CalCmsJobId           int
		LastCalCmsState       string
		LastMairListCommState string
		MairListPlaying       bool
		EventsPresent         int
		EventsMissing         int
		EventsMultiple        int
	}
}

var (
	EnvFile = ".env"
)

// InitConfig initializes the configuration and sets the defaults
func InitConfig(file string, config *AppConfig) error {
	log.Printf("Initializing configuration from file %v...", file)
	if err := loadConfig(file); err != nil {
		log.Printf("Error while loading configuration from file. %v", err)
	}
	if err := envconfig.Process("", config); err != nil {
		return fmt.Errorf("could not initialize configuration: %v", err.Error())
	}
	setDefaults(config)
	log.Print("Configuration initialized")
	return nil
}

// cleanFilePath does sanity-checking on file paths
func checkFilePath(filePath *string) {
	if *filePath != "" {
		*filePath = filepath.Clean(*filePath)
		_, err := os.Stat(*filePath)
		if err == nil {
			*filePath, err = filepath.EvalSymlinks(*filePath)
			if err != nil {
				log.Printf("error checking file %v", *filePath)
			}
		}
	}
}

// setDefaults sets defaults for some configurations items
func setDefaults(config *AppConfig) {
	config.RunTime.LastCalCmsState = "N/A"
	config.RunTime.LastMairListCommState = "N/A"
	if len(config.Crawl.StreamMap) == 0 {
		config.Crawl.StreamMap = make(map[string]int)
	}
	checkFilePath(&config.Server.CertFile)
	checkFilePath(&config.Server.KeyFile)
	checkFilePath(&config.Server.LogFile)
	checkFilePath(&config.Misc.FileSaveFile)
	checkFilePath(&config.Crawl.RootFolder)
	checkFilePath(&config.Crawl.FfprobePath)
	checkFilePath(&config.Export.ExportFolder)
}

// loadConfig loads the configuration from file. Returns an error if loading fails
func loadConfig(file string) error {
	if err := godotenv.Load(file); err != nil {
		return err
	}
	return nil
}
