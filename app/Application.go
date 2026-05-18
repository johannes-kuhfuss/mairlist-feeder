// package app ties together all bits and pieces to start the program
package app

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/robfig/cron/v3"

	metrics "github.com/johannes-kuhfuss/mairlist-feeder/Metrics"
	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/handlers"
	"github.com/johannes-kuhfuss/mairlist-feeder/repositories"
	"github.com/johannes-kuhfuss/mairlist-feeder/service"
	"github.com/johannes-kuhfuss/services_utils/date"
	"github.com/johannes-kuhfuss/services_utils/logger"
)

type Application struct {
	cfg            config.AppConfig
	server         http.Server
	appEnd         chan os.Signal
	appCtx         context.Context
	appCancel      context.CancelFunc
	statsUiHandler handlers.StatsUiHandler
	fileRepo       repositories.DefaultFileRepository
	crawlService   service.DefaultCrawlService
	cleanService   service.DefaultCleanService
	exportService  service.DefaultExportService
	calCmsService  service.DefaultCalCmsService
}

const (
	eventUrl  = "/events"
	fileUrl   = "/filelist"
	actionUrl = "/actions"
)

// StartApp orchestrates the startup of the application
func StartApp() {
	application := &Application{}
	application.Start()
}

func (a *Application) Start() {
	getCmdLine()
	err := config.InitConfig(config.EnvFile, &a.cfg)
	if err != nil {
		panic(err)
	}
	logger.Init(a.cfg.Server.LogFile)
	logger.Info("Starting application...")
	if a.cfg.Server.LogFile != "" {
		logger.Infof("Logging to file: %v", a.cfg.Server.LogFile)
	} else {
		logger.Info("Logging to file disabled")
	}
	a.initRouter()
	a.initServer()
	metrics.InitMetrics(&a.cfg, prometheus.DefaultRegisterer)
	a.wireApp()
	a.mapUrls()
	a.RegisterForOsSignals()
	a.appCtx, a.appCancel = context.WithCancel(context.Background())
	a.scheduleBgJobs()
	go a.startServer()
	if a.cfg.Export.QueryMairListStatus {
		go a.exportService.QueryStatus(a.appCtx)
	}
	a.crawlService.Crawl()
	if _, err := a.calCmsService.RefreshTodayEvents(); err != nil {
		logger.Error("Error refreshing today's events", err)
	}

	<-a.appEnd
	shutdownCtx, shutdownCancel := a.cleanUp()
	defer shutdownCancel()
	if err := a.server.Shutdown(shutdownCtx); err != nil {
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
func (a *Application) initRouter() {
	gin.SetMode(a.cfg.Gin.Mode)
	router := gin.New()
	if a.cfg.Gin.LogToLogger {
		gin.DefaultWriter = logger.GetLogger()
		router.Use(gin.Logger())
	}
	router.Use(gin.Recovery())
	router.SetTrustedProxies(nil)
	globPath := filepath.Join(a.cfg.Gin.TemplatePath, "*.tmpl")
	router.LoadHTMLGlob(globPath)

	a.cfg.RunTime.Router = router
}

// initServer checks whether https is enabled and initializes the web server accordingly
func (a *Application) initServer() {
	var tlsConfig tls.Config

	if a.cfg.Server.UseTls {
		tlsConfig = tls.Config{
			PreferServerCipherSuites: true,
			MinVersion:               tls.VersionTLS13,
			CurvePreferences: []tls.CurveID{
				tls.X25519,
				tls.CurveP256,
				tls.CurveP384,
			},
		}
	}
	if a.cfg.Server.UseTls {
		a.cfg.RunTime.ListenAddr = fmt.Sprintf("%s:%s", a.cfg.Server.Host, a.cfg.Server.TlsPort)
	} else {
		a.cfg.RunTime.ListenAddr = fmt.Sprintf("%s:%s", a.cfg.Server.Host, a.cfg.Server.Port)
	}

	a.server = http.Server{
		Addr:              a.cfg.RunTime.ListenAddr,
		Handler:           a.cfg.RunTime.Router,
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    http.DefaultMaxHeaderBytes,
	}
	if a.cfg.Server.UseTls {
		a.server.TLSConfig = &tlsConfig
		a.server.TLSNextProto = make(map[string]func(*http.Server, *tls.Conn, http.Handler))
	}
}

// wireApp initializes the services in the right order and injects the dependencies
func (a *Application) wireApp() {
	a.fileRepo = repositories.NewFileRepository(&a.cfg)
	a.calCmsService = service.NewCalCmsService(&a.cfg, &a.fileRepo)
	a.crawlService = service.NewCrawlService(&a.cfg, &a.fileRepo, a.calCmsService)
	a.cleanService = service.NewCleanService(&a.cfg, &a.fileRepo)
	a.exportService = service.NewExportService(&a.cfg, &a.fileRepo)
	a.statsUiHandler = handlers.NewStatsUiHandler(&a.cfg, &a.fileRepo, &a.crawlService, &a.exportService, &a.cleanService, &a.calCmsService)
}

// mapUrls defines the handlers for the available URLs
func (a *Application) mapUrls() {
	staticRoot, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(err)
	}
	a.cfg.RunTime.Router.StaticFS("/static", http.FS(staticRoot))

	a.cfg.RunTime.Router.GET("/", a.statsUiHandler.StatusPage)
	a.cfg.RunTime.Router.GET(fileUrl, a.statsUiHandler.FileListPage)
	a.cfg.RunTime.Router.GET(eventUrl, a.statsUiHandler.EventListPage)
	a.cfg.RunTime.Router.GET("/yesterday", a.statsUiHandler.YesterdaysEvents)
	a.cfg.RunTime.Router.GET(actionUrl, a.statsUiHandler.ActionPage)
	a.cfg.RunTime.Router.POST(actionUrl, a.statsUiHandler.ExecAction)
	a.cfg.RunTime.Router.GET("/logs", a.statsUiHandler.LogsPage)
	a.cfg.RunTime.Router.GET("/about", a.statsUiHandler.AboutPage)
	a.cfg.RunTime.Router.GET("/metrics", gin.WrapH(promhttp.Handler()))
}

// RegisterForOsSignals listens for OS signals terminating the program and sends an internal signal to start cleanup
func (a *Application) RegisterForOsSignals() {
	a.appEnd = make(chan os.Signal, 1)
	signal.Notify(a.appEnd, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
}

// scheduleBgJobs schedules all jobs running in the background, e.g. cleaning yesterday's items from the list
func (a *Application) scheduleBgJobs() {
	// cron format: Minutes, Hours, day of Month, Month, Day of Week
	logger.Info("Scheduling jobs...")
	crawlCycle := "@every " + strconv.Itoa(a.cfg.Crawl.CrawlCycleMin) + "m"
	a.cfg.RunTime.BgJobs = cron.New()
	// Crawl every x minutes
	crawlId, crawlErr := a.cfg.RunTime.BgJobs.AddFunc(crawlCycle, a.crawlService.Crawl)
	// Clean 00:30 local time
	cleanId, cleanErr := a.cfg.RunTime.BgJobs.AddFunc("30 0 * * *", a.cleanService.Clean)
	// Export every hour, x minutes to the hour
	exportStr := fmt.Sprintf("%02d * * * *", a.cfg.Export.ExportMinute)
	exportId, exportErr := a.cfg.RunTime.BgJobs.AddFunc(exportStr, a.exportService.Export)
	a.cfg.RunTime.BgJobs.Start()
	if crawlErr != nil {
		logger.Errorf("Error when scheduling job %v for crawling. %v", crawlId, crawlErr)
	} else {
		a.cfg.RunTime.CrawlJobId = crawlId
		logger.Infof("Crawl Job: %v - Next execution: %v", a.cfg.RunTime.BgJobs.Entry(crawlId).Job, a.cfg.RunTime.BgJobs.Entry(crawlId).Next.String())
	}
	if cleanErr != nil {
		logger.Errorf("Error when scheduling job %v for cleaning. %v", cleanId, cleanErr)
	} else {
		a.cfg.RunTime.CleanJobId = cleanId
		logger.Infof("Clean Job: %v - Next execution: %v", a.cfg.RunTime.BgJobs.Entry(cleanId).Job, a.cfg.RunTime.BgJobs.Entry(cleanId).Next.String())
	}
	if exportErr != nil {
		logger.Errorf("Error when scheduling job %v for exporting. %v", exportId, exportErr)
	} else {
		a.cfg.RunTime.ExportJobId = exportId
		logger.Infof("Export Job: %v - Next execution: %v", a.cfg.RunTime.BgJobs.Entry(exportId).Job, a.cfg.RunTime.BgJobs.Entry(exportId).Next.String())
	}
	// Export day's events at 23:15 (before we start looking at the next day)
	if a.cfg.CalCms.ExportDayEvents {
		eventId, eventErr := a.cfg.RunTime.BgJobs.AddFunc("15 23 * * *", a.ExportDayDataRun)
		if eventErr != nil {
			logger.Errorf("Error when scheduling job %v for recording day's events state. %v", eventId, eventErr)
		} else {
			a.cfg.RunTime.EventJobId = eventId
			logger.Infof("Recording Day's Events Job: %v - Next execution: %v", a.cfg.RunTime.BgJobs.Entry(eventId).Job, a.cfg.RunTime.BgJobs.Entry(eventId).Next.String())
		}
	}
	if a.cfg.CalCms.QueryCalCms {
		calCmsId, calCmsErr := a.cfg.RunTime.BgJobs.AddFunc("@every 1m", a.calCmsService.CountRun)
		if calCmsErr != nil {
			logger.Errorf("Error when scheduling job %v for CalCMS event counting. %v", calCmsId, calCmsErr)
		} else {
			a.cfg.RunTime.CalCmsJobId = calCmsId
			logger.Infof("CalCMS Event Counting Job: %v - Next execution: %v", a.cfg.RunTime.BgJobs.Entry(calCmsId).Job, a.cfg.RunTime.BgJobs.Entry(calCmsId).Next.String())
		}
	}
	logger.Info("Jobs scheduled")
}

// startServer starts the preconfigured web server
func (a *Application) startServer() {
	logger.Infof("Listening on %v", a.cfg.RunTime.ListenAddr)
	a.cfg.RunTime.StartDate = date.GetNowUtc()
	if a.cfg.Server.UseTls {
		if err := a.server.ListenAndServeTLS(a.cfg.Server.CertFile, a.cfg.Server.KeyFile); err != nil && err != http.ErrServerClosed {
			logger.Error("Error while starting https server", err)
			panic(err)
		}
	} else {
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Error while starting http server", err)
			panic(err)
		}
	}
}

// cleanUp tries to clean up when the program is stopped and returns the shutdown timeout context.
func (a *Application) cleanUp() (context.Context, context.CancelFunc) {
	logger.Info("Cleaning up...")
	if a.appCancel != nil {
		a.appCancel()
	}
	shutdownTime := time.Duration(a.cfg.Server.GracefulShutdownTime) * time.Second
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTime)
	if a.cfg.RunTime.BgJobs != nil {
		cronCtx := a.cfg.RunTime.BgJobs.Stop()
		select {
		case <-cronCtx.Done():
			logger.Info("Background jobs stopped")
		case <-shutdownCtx.Done():
			logger.Warn("Timed out waiting for background jobs to stop")
		}
	}
	defer func() {
		logger.Info("Cleaned up")
	}()
	return shutdownCtx, shutdownCancel
}
