package config

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/johannes-kuhfuss/services_utils/api_error"
	"github.com/johannes-kuhfuss/services_utils/logger"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/robfig/cron/v3"
)

type AppConfig struct {
	Server struct {
		Host                 string `envconfig:"SERVER_HOST"`
		Port                 string `envconfig:"SERVER_PORT" default:"8080"`
		TlsPort              string `envconfig:"SERVER_TLS_PORT" default:"8443"`
		GracefulShutdownTime int    `envconfig:"GRACEFUL_SHUTDOWN_TIME" default:"10"`
		UseTls               bool   `envconfig:"USE_TLS" default:"false"`
		CertFile             string `envconfig:"CERT_FILE" default:"./cert/cert.pem"`
		KeyFile              string `envconfig:"KEY_FILE" default:"./cert/cert.key"`
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
		RootFolder              string         `envconfig:"ROOT_FOLDER" default:"Z:\\sendungen"`
		CrawlExtensions         []string       `envconfig:"CRAWL_EXTENSIONS" default:".mp3,.m4a,.wav,.stream"`
		AudioFileExtensions     []string       `envconfig:"AUDIO_FILE_EXTENSIONS" default:".mp3,.m4a,.wav"`
		StreamingFileExtensions []string       `envconfig:"STREAM_FILE_EXTENSIONS" default:".stream"`
		FfprobePath             string         `envconfig:"FFPROBE_PATH" default:"/usr/bin/ffprobe"`
		FfProbeTimeOut          int            `envconfig:"FFPROBE_TIMEOUT" default:"60"`
		CrawlCycleMin           int            `envconfig:"CRAWL_CYCLE_MIN" default:"15"`
		StreamMap               map[string]int `envconfig:"STREAM_MAP"`
	}
	Export struct {
		ExportFolder           string  `envconfig:"EXPORT_FOLDER" default:"C:\\TEMP"`
		ShortDeltaAllowance    float64 `envconfig:"SHORT_DELTA_ALLOWANCE" default:"5.0"`
		LongDeltaAllowance     float64 `envconfig:"LONG_DELTA_ALLOWANCE" default:"8.0"`
		MairListUrl            string  `envconfig:"MAIRLIST_URL" default:"http://localhost:9300/"`
		MairListUser           string  `envconfig:"MAIRLIST_USER" default:"dbtest"`
		MairListPassword       string  `envconfig:"MAIRLIST_PASS" default:"dbtest"`
		AppendPlaylist         bool    `envconfig:"APPEND_PLAYLIST" default:"false"`
		TerminateAfterDuration bool    `envconfig:"TERM_AFTER_DUR" default:"true"`
		LimitTime              bool    `envconfig:"LIMIT_TIME" default:"false"`
	}
	CalCms struct {
		QueryCalCms bool   `envconfig:"QUERY_CALCMS" default:"false"`
		CmsUrl      string `envconfig:"CALCMS_URL" default:"https://programm.coloradio.org/agenda/events.cgi"`
		Template    string `envconfig:"CALCMS_TEMPLATE" default:"event.json-p"`
	}
	RunTime struct {
		Router               *gin.Engine
		BgJobs               *cron.Cron
		ListenAddr           string
		StartDate            time.Time
		CrawlRunNumber       int
		LastCrawlDate        time.Time
		FilesInList          int
		AudioFilesInList     int
		StreamFilesInList    int
		LastExportRunDate    time.Time
		LastExportedFileDate time.Time
		LastExportFileName   string
		CrawlRunning         bool
		ExportRunning        bool
		CleanRunning         bool
		LastCleanDate        time.Time
		FilesCleaned         int
		CrawlJobId           int
		ExportJobId          int
		CleanJobId           int
	}
}

var (
	EnvFile = ".env"
)

func InitConfig(file string, config *AppConfig) api_error.ApiErr {
	logger.Info(fmt.Sprintf("Initalizing configuration from file %v", file))
	err := loadConfig(file)
	if err != nil {
		logger.Error("Error while loading config file: ", err)
	}
	err = envconfig.Process("", config)
	if err != nil {
		return api_error.NewInternalServerError("Could not initalize configuration: ", err)
	}
	setDefaults(config)
	logger.Info("Done initalizing configuration")
	return nil
}

func setDefaults(config *AppConfig) {
	config.RunTime.CrawlRunNumber = 0
	config.RunTime.CrawlRunning = false
	config.RunTime.ExportRunning = false
	config.RunTime.CleanRunning = false
}

func loadConfig(file string) error {
	err := godotenv.Load(file)
	if err != nil {
		return err
	}
	return nil
}
