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
	if err := application.Start(); err != nil {
		panic(err)
	}
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
	a.wireApp()
	if err := a.mapUrls(); err != nil {
		return err
	}
	a.RegisterForOsSignals()
	a.scheduleBgJobs()
	go a.startServer()
	if a.cfg.Export.QueryMairListStatus {
		go a.exportService.QueryStatus(a.appCtx)
	}
	if err := a.crawlService.Crawl(); err != nil {
		logger.Error("Error running initial crawl", err)
	}
	if _, err := a.calCmsService.RefreshTodayEvents(); err != nil {
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
		a.state.Runtime.ListenAddr = fmt.Sprintf("%s:%s", a.cfg.Server.Host, a.cfg.Server.TlsPort)
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
	if a.cfg.Server.UseTls {
		a.server.TLSConfig = &tlsConfig
		a.server.TLSNextProto = make(map[string]func(*http.Server, *tls.Conn, http.Handler))
	}
}

// wireApp initializes the services in the right order and injects the dependencies
func (a *Application) wireApp() {
	a.fileRepo = repositories.NewFileRepository(&a.cfg)
	a.calCmsService = service.NewCalCmsServiceWithState(&a.cfg, a.state, &a.fileRepo)
	a.crawlService = service.NewCrawlServiceWithState(&a.cfg, a.state, &a.fileRepo, a.calCmsService)
	a.cleanService = service.NewCleanServiceWithState(&a.cfg, a.state, &a.fileRepo)
	a.exportService = service.NewExportServiceWithState(&a.cfg, a.state, &a.fileRepo)
	a.statsUiHandler = handlers.NewStatsUiHandlerWithState(&a.cfg, a.state, &a.fileRepo, &a.crawlService, &a.exportService, &a.cleanService, &a.calCmsService)
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
	a.state.Runtime.BgJobs = cron.New()
	// Crawl every x minutes
	crawlId, crawlErr := a.state.Runtime.BgJobs.AddFunc(crawlCycle, func() {
		if err := a.crawlService.Crawl(); err != nil {
			logger.Error("Error running scheduled crawl", err)
		}
	})
	// Clean 00:30 local time
	cleanId, cleanErr := a.state.Runtime.BgJobs.AddFunc("30 0 * * *", func() {
		if err := a.cleanService.Clean(); err != nil {
			logger.Error("Error running scheduled clean", err)
		}
	})
	// Export every hour, x minutes to the hour
	exportStr := fmt.Sprintf("%02d * * * *", a.cfg.Export.ExportMinute)
	exportId, exportErr := a.state.Runtime.BgJobs.AddFunc(exportStr, func() {
		if err := a.exportService.Export(); err != nil {
			logger.Error("Error running scheduled export", err)
		}
	})
	a.state.Runtime.BgJobs.Start()
	if crawlErr != nil {
		logger.Errorf("Error when scheduling job %v for crawling. %v", crawlId, crawlErr)
	} else {
		a.state.Runtime.CrawlJobId = crawlId
		logger.Infof("Crawl Job: %v - Next execution: %v", a.state.Runtime.BgJobs.Entry(crawlId).Job, a.state.Runtime.BgJobs.Entry(crawlId).Next.String())
	}
	if cleanErr != nil {
		logger.Errorf("Error when scheduling job %v for cleaning. %v", cleanId, cleanErr)
	} else {
		a.state.Runtime.CleanJobId = cleanId
		logger.Infof("Clean Job: %v - Next execution: %v", a.state.Runtime.BgJobs.Entry(cleanId).Job, a.state.Runtime.BgJobs.Entry(cleanId).Next.String())
	}
	if exportErr != nil {
		logger.Errorf("Error when scheduling job %v for exporting. %v", exportId, exportErr)
	} else {
		a.state.Runtime.ExportJobId = exportId
		logger.Infof("Export Job: %v - Next execution: %v", a.state.Runtime.BgJobs.Entry(exportId).Job, a.state.Runtime.BgJobs.Entry(exportId).Next.String())
	}
	// Export day's events at 23:15 (before we start looking at the next day)
	if a.cfg.CalCms.ExportDayEvents {
		eventId, eventErr := a.state.Runtime.BgJobs.AddFunc("15 23 * * *", a.ExportDayDataRun)
		if eventErr != nil {
			logger.Errorf("Error when scheduling job %v for recording day's events state. %v", eventId, eventErr)
		} else {
			a.state.Runtime.EventJobId = eventId
			logger.Infof("Recording Day's Events Job: %v - Next execution: %v", a.state.Runtime.BgJobs.Entry(eventId).Job, a.state.Runtime.BgJobs.Entry(eventId).Next.String())
		}
	}
	if a.cfg.CalCms.QueryCalCms {
		calCmsId, calCmsErr := a.state.Runtime.BgJobs.AddFunc("@every 1m", a.calCmsService.CountRun)
		if calCmsErr != nil {
			logger.Errorf("Error when scheduling job %v for CalCMS event counting. %v", calCmsId, calCmsErr)
		} else {
			a.state.Runtime.CalCmsJobId = calCmsId
			logger.Infof("CalCMS Event Counting Job: %v - Next execution: %v", a.state.Runtime.BgJobs.Entry(calCmsId).Job, a.state.Runtime.BgJobs.Entry(calCmsId).Next.String())
		}
	}
	logger.Info("Jobs scheduled")
}

// startServer starts the preconfigured web server
func (a *Application) startServer() {
	logger.Infof("Listening on %v", a.state.Runtime.ListenAddr)
	a.state.Runtime.StartDate = date.GetNowUtc()
	if a.cfg.Server.UseTls {
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
	if a.state.Runtime.BgJobs != nil {
		cronCtx := a.state.Runtime.BgJobs.Stop()
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
