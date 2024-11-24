// package app ties together all bits and pieces to start the program
package app

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/robfig/cron/v3"

	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/handlers"
	"github.com/johannes-kuhfuss/mairlist-feeder/repositories"
	"github.com/johannes-kuhfuss/mairlist-feeder/service"
	"github.com/johannes-kuhfuss/services_utils/date"
	"github.com/johannes-kuhfuss/services_utils/logger"
)

var (
	cfg            config.AppConfig
	server         http.Server
	appEnd         chan os.Signal
	ctx            context.Context
	cancel         context.CancelFunc
	statsUiHandler handlers.StatsUiHandler
	fileRepo       repositories.DefaultFileRepository
	crawlService   service.DefaultCrawlService
	cleanService   service.DefaultCleanService
	exportService  service.DefaultExportService
	calCmsService  service.DefaultCalCmsService
)

// StartApp orchestrates the startup of the application
func StartApp() {
	getCmdLine()
	err := config.InitConfig(config.EnvFile, &cfg)
	if err != nil {
		panic(err)
	}
	logger.Init(cfg.Server.LogFile)
	logger.Info("Starting application...")
	if cfg.Server.LogFile != "" {
		logger.Infof("Logging to file: %v", cfg.Server.LogFile)
	} else {
		logger.Info("Logging to file disabled")
	}
	initRouter()
	initServer()
	initMetrics()
	wireApp()
	mapUrls()
	RegisterForOsSignals()
	scheduleBgJobs()
	go startServer()
	if cfg.Export.QueryMairListStatus {
		go exportService.QueryStatus()
	}
	go updateMetrics()
	crawlService.Crawl()
	calCmsService.GetEvents()

	<-appEnd
	cleanUp()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Graceful shutdown failed", err)
	} else {
		logger.Info("Graceful shutdown finished")
	}
}

// getCmdLine checks the command line arguments
func getCmdLine() {
	flag.StringVar(&config.EnvFile, "config.file", ".env", "Specify location of config file. Default is .env")
	flag.Parse()
}

// initRouter initializes gin-gonic as the router
func initRouter() {
	gin.SetMode(cfg.Gin.Mode)
	router := gin.New()
	if cfg.Gin.LogToLogger {
		gin.DefaultWriter = logger.GetLogger()
		router.Use(gin.Logger())
	}
	router.Use(gin.Recovery())
	router.SetTrustedProxies(nil)
	globPath := filepath.Join(cfg.Gin.TemplatePath, "*.tmpl")
	router.LoadHTMLGlob(globPath)

	cfg.RunTime.Router = router
}

// initServer checks whether https is enabled and initializes the web server accordingly
func initServer() {
	var tlsConfig tls.Config

	if cfg.Server.UseTls {
		tlsConfig = tls.Config{
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			},
			PreferServerCipherSuites: true,
			MinVersion:               tls.VersionTLS12,
			CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		}
	}
	if cfg.Server.UseTls {
		cfg.RunTime.ListenAddr = fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.TlsPort)
	} else {
		cfg.RunTime.ListenAddr = fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	}

	server = http.Server{
		Addr:              cfg.RunTime.ListenAddr,
		Handler:           cfg.RunTime.Router,
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 0,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    0,
	}
	if cfg.Server.UseTls {
		server.TLSConfig = &tlsConfig
		server.TLSNextProto = make(map[string]func(*http.Server, *tls.Conn, http.Handler))
	}
}

// wireApp initializes the services in the right order and injects the dependencies
func wireApp() {
	fileRepo = repositories.NewFileRepository(&cfg)
	calCmsService = service.NewCalCmsService(&cfg, &fileRepo)
	crawlService = service.NewCrawlService(&cfg, &fileRepo, calCmsService)
	cleanService = service.NewCleanService(&cfg, &fileRepo)
	exportService = service.NewExportService(&cfg, &fileRepo)
	statsUiHandler = handlers.NewStatsUiHandler(&cfg, &fileRepo, &crawlService, &exportService, &cleanService, &calCmsService)
}

// mapUrls defines the handlers for the available URLs
func mapUrls() {
	cfg.RunTime.Router.GET("/", statsUiHandler.StatusPage)
	cfg.RunTime.Router.GET("/filelist", statsUiHandler.FileListPage)
	cfg.RunTime.Router.GET("/events", statsUiHandler.EventListPage)
	cfg.RunTime.Router.GET("/actions", statsUiHandler.ActionPage)
	cfg.RunTime.Router.POST("/actions", statsUiHandler.ExecAction)
	cfg.RunTime.Router.GET("/logs", statsUiHandler.LogsPage)
	cfg.RunTime.Router.GET("/about", statsUiHandler.AboutPage)
	cfg.RunTime.Router.GET("/metrics", gin.WrapH(promhttp.Handler()))
}

// RegisterForOsSignals listens for OS signals terminating the program and sends an internal signal to start cleanup
func RegisterForOsSignals() {
	appEnd = make(chan os.Signal, 1)
	signal.Notify(appEnd, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
}

// scheduleBgJobs schedules all jobs running in the background, e.g. cleaning yesterday's items from the list
func scheduleBgJobs() {
	// cron format: Minutes, Hours, day of Month, Month, Day of Week
	logger.Info("Scheduling jobs...")
	crawlCycle := "@every " + strconv.Itoa(cfg.Crawl.CrawlCycleMin) + "m"
	cfg.RunTime.BgJobs = cron.New()
	// Crawl every x minutes
	crawlId, crawlErr := cfg.RunTime.BgJobs.AddFunc(crawlCycle, crawlService.Crawl)
	// Clean 00:30 local time
	cleanId, cleanErr := cfg.RunTime.BgJobs.AddFunc("30 0 * * *", cleanService.Clean)
	// Export every hour, 10 minutes to the hour
	exportId, exportErr := cfg.RunTime.BgJobs.AddFunc("50 * * * *", exportService.Export)
	cfg.RunTime.BgJobs.Start()
	if crawlErr != nil {
		logger.Errorf("Error when scheduling job %v for crawling. %v", crawlId, crawlErr)
	} else {
		cfg.RunTime.CrawlJobId = int(crawlId)
		logger.Infof("Crawl Job: %v - Next execution: %v", cfg.RunTime.BgJobs.Entry(crawlId).Job, cfg.RunTime.BgJobs.Entry(crawlId).Next.String())
	}
	if cleanErr != nil {
		logger.Errorf("Error when scheduling job %v for cleaning. %v", cleanId, cleanErr)
	} else {
		cfg.RunTime.CleanJobId = int(cleanId)
		logger.Infof("Clean Job: %v - Next execution: %v", cfg.RunTime.BgJobs.Entry(cleanId).Job, cfg.RunTime.BgJobs.Entry(cleanId).Next.String())
	}
	if exportErr != nil {
		logger.Errorf("Error when scheduling job %v for exporting. %v", exportId, exportErr)
	} else {
		cfg.RunTime.ExportJobId = int(exportId)
		logger.Infof("Export Job: %v - Next execution: %v", cfg.RunTime.BgJobs.Entry(exportId).Job, cfg.RunTime.BgJobs.Entry(exportId).Next.String())
	}
	// Export day's events at 23:15 (before we start looking at the next day)
	if cfg.CalCms.ExportDayEvents {
		eventId, eventErr := cfg.RunTime.BgJobs.AddFunc("15 23 * * *", ExportDayDataRun)
		if eventErr != nil {
			logger.Errorf("Error when scheduling job %v for recording day's events state. %v", eventId, eventErr)
		} else {
			cfg.RunTime.EventJobId = int(eventId)
			logger.Infof("Recording Day's Events Job: %v - Next execution: %v", cfg.RunTime.BgJobs.Entry(eventId).Job, cfg.RunTime.BgJobs.Entry(eventId).Next.String())
		}
	}
	if cfg.CalCms.QueryCalCms {
		calCmsId, calCmsErr := cfg.RunTime.BgJobs.AddFunc("@every 1m", calCmsService.CountRun)
		if calCmsErr != nil {
			logger.Errorf("Error when scheduling job %v for CalCMS event counting. %v", calCmsId, calCmsErr)
		} else {
			cfg.RunTime.CalCmsJobId = int(calCmsId)
			logger.Infof("CalCMS Event Counting Job: %v - Next execution: %v", cfg.RunTime.BgJobs.Entry(calCmsId).Job, cfg.RunTime.BgJobs.Entry(calCmsId).Next.String())
		}
	}
	logger.Info("Jobs scheduled")
}

// startServer starts the preconfigured web server
func startServer() {
	logger.Infof("Listening on %v", cfg.RunTime.ListenAddr)
	cfg.RunTime.StartDate = date.GetNowUtc()
	if cfg.Server.UseTls {
		if err := server.ListenAndServeTLS(cfg.Server.CertFile, cfg.Server.KeyFile); err != nil && err != http.ErrServerClosed {
			logger.Error("Error while starting https server", err)
			panic(err)
		}
	} else {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Error while starting http server", err)
			panic(err)
		}
	}
}

// cleanUp tries to clean up when the program is stopped
func cleanUp() {
	logger.Info("Cleaning up...")
	cfg.RunTime.BgJobs.Stop()
	shutdownTime := time.Duration(cfg.Server.GracefulShutdownTime) * time.Second
	ctx, cancel = context.WithTimeout(context.Background(), shutdownTime)
	defer func() {
		logger.Info("Cleaned up")
		cancel()
	}()
}
