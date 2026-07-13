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

	"github.com/johannes-kuhfuss/mairlist-feeder/appstate"
	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/dto"
	"github.com/johannes-kuhfuss/mairlist-feeder/handlers"
	metrics "github.com/johannes-kuhfuss/mairlist-feeder/metrics"
	"github.com/johannes-kuhfuss/mairlist-feeder/repositories"
	"github.com/johannes-kuhfuss/mairlist-feeder/service"
	"github.com/johannes-kuhfuss/services_utils/date"
	"github.com/johannes-kuhfuss/services_utils/logger"
)

type Application struct {
	cfg            config.AppConfig
	state          *appstate.AppState
	server         http.Server
	appCtx         context.Context
	appCancel      context.CancelFunc
	statsUiHandler handlers.StatsUiHandler
	fileRepo       repositories.FileRepository
	crawlService   applicationCrawler
	cleanService   applicationCleaner
	exportService  applicationExporter
	calCmsService  applicationCalCms
}

type applicationCrawler interface {
	service.Crawler
}

type applicationCleaner interface {
	service.Cleaner
}

type applicationExporter interface {
	service.Exporter
	ExportAllHoursContext(context.Context) error
	ExportForHourContext(context.Context, string) error
	QueryStatus(context.Context)
}

type applicationCalCms interface {
	service.CalCmsQuerier
	RefreshTodayEventsContext(context.Context) ([]dto.Event, error)
	GetTodayEvents() ([]dto.Event, error)
	GetYesterdaysEvents() []dto.Event
	SaveYesterdaysEvents()
	CountRunContext(context.Context)
}

const (
	eventUrl  = "/events"
	fileUrl   = "/filelist"
	actionUrl = "/actions"
)

// StartApp orchestrates the startup of the application
func StartApp() error {
	application := &Application{}
	return application.Start()
}

func (a *Application) Start() error {
	a.state = appstate.New()
	getCmdLine()
	err := config.InitConfig(config.EnvFile, &a.cfg)
	if err != nil {
		return err
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
	metrics.InitMetrics(a.state, prometheus.DefaultRegisterer)
	a.RegisterForOsSignals()
	a.wireApp()
	if err := a.mapUrls(); err != nil {
		return err
	}
	a.scheduleBgJobs()
	go a.startServer()
	if a.cfg.Export.QueryMairListStatus {
		go a.exportService.QueryStatus(a.appCtx)
	}
	if err := a.crawlService.CrawlContext(a.appCtx); err != nil {
		logger.Error("Error running initial crawl", err)
	}
	if _, err := a.calCmsService.RefreshTodayEventsContext(a.appCtx); err != nil {
		logger.Error("Error refreshing today's events", err)
	}

	<-a.appCtx.Done()
	shutdownCtx, shutdownCancel := a.cleanUp()
	defer shutdownCancel()
	if err := a.server.Shutdown(shutdownCtx); err != nil {
		logger.Error("Graceful shutdown failed", err)
	} else {
		logger.Info("Graceful shutdown finished")
	}
	return nil
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

	a.state.Runtime.Router = router
}

// initServer checks whether https is enabled and initializes the web server accordingly
func (a *Application) initServer() {
	var tlsConfig tls.Config

	if a.cfg.Server.UseTLS {
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
	if a.cfg.Server.UseTLS {
		a.state.Runtime.ListenAddr = fmt.Sprintf("%s:%s", a.cfg.Server.Host, a.cfg.Server.TLSPort)
	} else {
		a.state.Runtime.ListenAddr = fmt.Sprintf("%s:%s", a.cfg.Server.Host, a.cfg.Server.Port)
	}

	a.server = http.Server{
		Addr:              a.state.Runtime.ListenAddr,
		Handler:           a.state.Runtime.Router,
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    http.DefaultMaxHeaderBytes,
	}
	if a.cfg.Server.UseTLS {
		a.server.TLSConfig = &tlsConfig
		a.server.TLSNextProto = make(map[string]func(*http.Server, *tls.Conn, http.Handler))
	}
}

// wireApp initializes the services in the right order and injects the dependencies
func (a *Application) wireApp() {
	fileRepo := repositories.NewFileRepository(&a.cfg)
	calCmsService := service.NewCalCmsServiceWithState(&a.cfg, a.state, &fileRepo)
	crawlService := service.NewCrawlServiceWithState(&a.cfg, a.state, &fileRepo, &calCmsService)
	cleanService := service.NewCleanServiceWithState(&a.cfg, a.state, &fileRepo)
	exportService := service.NewExportServiceWithState(&a.cfg, a.state, &fileRepo)
	a.fileRepo = &fileRepo
	a.calCmsService = &calCmsService
	a.crawlService = &crawlService
	a.cleanService = &cleanService
	a.exportService = &exportService
	a.statsUiHandler = handlers.NewStatsUiHandlerWithContext(a.appCtx, &a.cfg, a.state, a.fileRepo, a.crawlService, a.exportService, a.cleanService, a.calCmsService)
}

// mapUrls defines the handlers for the available URLs
func (a *Application) mapUrls() error {
	staticRoot, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return err
	}
	a.state.Runtime.Router.StaticFS("/static", http.FS(staticRoot))

	a.state.Runtime.Router.GET("/", a.statsUiHandler.StatusPage)
	a.state.Runtime.Router.GET(fileUrl, a.statsUiHandler.FileListPage)
	a.state.Runtime.Router.GET(eventUrl, a.statsUiHandler.EventListPage)
	a.state.Runtime.Router.GET("/yesterday", a.statsUiHandler.YesterdaysEvents)
	a.state.Runtime.Router.GET(actionUrl, a.statsUiHandler.ActionPage)
	a.state.Runtime.Router.POST(actionUrl, a.statsUiHandler.ExecAction)
	a.state.Runtime.Router.GET(actionUrl+"/:id", a.statsUiHandler.ActionStatus)
	a.state.Runtime.Router.GET("/logs", a.statsUiHandler.LogsPage)
	a.state.Runtime.Router.GET("/about", a.statsUiHandler.AboutPage)
	a.state.Runtime.Router.GET("/healthz", a.healthz)
	a.state.Runtime.Router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	return nil
}

func (a *Application) healthz(c *gin.Context) {
	if a.cfg.Crawl.RootFolder != "" {
		if info, err := os.Stat(a.cfg.Crawl.RootFolder); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "root folder is not accessible"})
			return
		} else if !info.IsDir() {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "root folder is not a directory"})
			return
		}
	}
	if a.cfg.Export.ExportFolder != "" {
		if info, err := os.Stat(a.cfg.Export.ExportFolder); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "export folder is not accessible"})
			return
		} else if !info.IsDir() {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "export folder is not a directory"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// RegisterForOsSignals listens for OS signals terminating the program and sends an internal signal to start cleanup
func (a *Application) RegisterForOsSignals() {
	a.appCtx, a.appCancel = signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
}

// scheduleBgJobs schedules all jobs running in the background, e.g. cleaning yesterday's items from the list
func (a *Application) scheduleBgJobs() {
	// cron format: Minutes, Hours, day of Month, Month, Day of Week
	logger.Info("Scheduling jobs...")
	crawlCycle := "@every " + strconv.Itoa(a.cfg.Crawl.CrawlCycleMin) + "m"
	bgJobs := cron.New()
	a.state.Runtime.Update(func(runtime *appstate.RuntimeState) { runtime.BgJobs = bgJobs })
	// Crawl every x minutes
	crawlID, crawlErr := bgJobs.AddFunc(crawlCycle, func() {
		if err := a.crawlService.CrawlContext(a.appCtx); err != nil {
			logger.Error("Error running scheduled crawl", err)
		}
	})
	// Clean 00:30 local time
	cleanID, cleanErr := bgJobs.AddFunc("30 0 * * *", func() {
		if err := a.cleanService.CleanContext(a.appCtx); err != nil {
			logger.Error("Error running scheduled clean", err)
		}
	})
	// Export every hour, x minutes to the hour
	exportStr := fmt.Sprintf("%02d * * * *", a.cfg.Export.ExportMinute)
	exportID, exportErr := bgJobs.AddFunc(exportStr, func() {
		if err := a.exportService.ExportContext(a.appCtx); err != nil {
			logger.Error("Error running scheduled export", err)
		}
	})
	if crawlErr != nil {
		logger.Errorf("Error when scheduling job %v for crawling. %v", crawlID, crawlErr)
	} else {
		a.state.Runtime.Update(func(runtime *appstate.RuntimeState) { runtime.CrawlJobID = crawlID })
		logger.Infof("Crawl Job: %v", bgJobs.Entry(crawlID).Job)
	}
	if cleanErr != nil {
		logger.Errorf("Error when scheduling job %v for cleaning. %v", cleanID, cleanErr)
	} else {
		a.state.Runtime.Update(func(runtime *appstate.RuntimeState) { runtime.CleanJobID = cleanID })
		logger.Infof("Clean Job: %v", bgJobs.Entry(cleanID).Job)
	}
	if exportErr != nil {
		logger.Errorf("Error when scheduling job %v for exporting. %v", exportID, exportErr)
	} else {
		a.state.Runtime.Update(func(runtime *appstate.RuntimeState) { runtime.ExportJobID = exportID })
		logger.Infof("Export Job: %v", bgJobs.Entry(exportID).Job)
	}
	// Export day's events at 23:15 (before we start looking at the next day)
	if a.cfg.CalCms.ExportDayEvents {
		eventID, eventErr := bgJobs.AddFunc("15 23 * * *", a.ExportDayDataRun)
		if eventErr != nil {
			logger.Errorf("Error when scheduling job %v for recording day's events state. %v", eventID, eventErr)
		} else {
			a.state.Runtime.Update(func(runtime *appstate.RuntimeState) { runtime.EventJobID = eventID })
			logger.Infof("Recording Day's Events Job: %v", bgJobs.Entry(eventID).Job)
		}
	}
	if a.cfg.CalCms.QueryCalCms {
		calCmsID, calCmsErr := bgJobs.AddFunc("@every 1m", func() { a.calCmsService.CountRunContext(a.appCtx) })
		if calCmsErr != nil {
			logger.Errorf("Error when scheduling job %v for CalCMS event counting. %v", calCmsID, calCmsErr)
		} else {
			a.state.Runtime.Update(func(runtime *appstate.RuntimeState) { runtime.CalCmsJobID = calCmsID })
			logger.Infof("CalCMS Event Counting Job: %v", bgJobs.Entry(calCmsID).Job)
		}
	}
	bgJobs.Start()
	logger.Info("Jobs scheduled")
}

// startServer starts the preconfigured web server
func (a *Application) startServer() {
	runtime := a.state.Runtime.Snapshot()
	logger.Infof("Listening on %v", runtime.ListenAddr)
	a.state.Runtime.Update(func(runtime *appstate.RuntimeState) { runtime.StartDate = date.GetNowUtc() })
	if a.cfg.Server.UseTLS {
		if err := a.server.ListenAndServeTLS(a.cfg.Server.CertFile, a.cfg.Server.KeyFile); err != nil && err != http.ErrServerClosed {
			logger.Error("Error while starting https server", err)
			a.appCancel()
		}
	} else {
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Error while starting http server", err)
			a.appCancel()
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
	if bgJobs := a.state.Runtime.Snapshot().BgJobs; bgJobs != nil {
		cronCtx := bgJobs.Stop()
		select {
		case <-cronCtx.Done():
			logger.Info("Background jobs stopped")
		case <-shutdownCtx.Done():
			logger.Warn("Timed out waiting for background jobs to stop")
		}
	}
	a.statsUiHandler.Close()
	defer func() {
		logger.Info("Cleaned up")
	}()
	return shutdownCtx, shutdownCancel
}
