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
	calCmsService  service.CalCmsService
)

func StartApp() {
	logger.Info("Starting application")

	getCmdLine()
	err := config.InitConfig(config.EnvFile, &cfg)
	if err != nil {
		panic(err)
	}
	initRouter()
	initServer()
	wireApp()
	mapUrls()
	RegisterForOsSignals()
	scheduleBgJobs()
	go startServer()
	crawlService.Crawl()

	<-appEnd
	cleanUp()

	if srvErr := server.Shutdown(ctx); srvErr != nil {
		logger.Error("Graceful shutdown failed", srvErr)
	} else {
		logger.Info("Graceful shutdown finished")
	}
}

func getCmdLine() {
	flag.StringVar(&config.EnvFile, "config.file", ".env", "Specify location of config file. Default is .env")
	flag.Parse()
}

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

func wireApp() {
	fileRepo = repositories.NewFileRepository(&cfg)
	calCmsService = service.NewCalCmsService(&cfg, &fileRepo)
	crawlService = service.NewCrawlService(&cfg, &fileRepo, calCmsService)
	cleanService = service.NewCleanService(&cfg, &fileRepo)
	exportService = service.NewExportService(&cfg, &fileRepo)
	statsUiHandler = handlers.NewStatsUiHandler(&cfg, &fileRepo, &crawlService, &exportService, &cleanService)
}

func mapUrls() {
	cfg.RunTime.Router.GET("/", statsUiHandler.StatusPage)
	cfg.RunTime.Router.GET("/filelist", statsUiHandler.FileListPage)
	cfg.RunTime.Router.GET("/actions", statsUiHandler.ActionPage)
	cfg.RunTime.Router.POST("/actions", statsUiHandler.ExecAction)
	cfg.RunTime.Router.GET("/logs", statsUiHandler.LogsPage)
	cfg.RunTime.Router.GET("/about", statsUiHandler.AboutPage)
}

func RegisterForOsSignals() {
	appEnd = make(chan os.Signal, 1)
	signal.Notify(appEnd, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
}

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
		logger.Error(fmt.Sprintf("Error when scheduling job %v for crawling", crawlId), crawlErr)
	} else {
		cfg.RunTime.CrawlJobId = int(crawlId)
		logger.Info(fmt.Sprintf("Crawl Job: %v - Next execution: %v", cfg.RunTime.BgJobs.Entry(crawlId).Job, cfg.RunTime.BgJobs.Entry(crawlId).Next.String()))
	}
	if cleanErr != nil {
		logger.Error(fmt.Sprintf("Error when scheduling job %v for cleaning", cleanId), cleanErr)
	} else {
		cfg.RunTime.CleanJobId = int(cleanId)
		logger.Info(fmt.Sprintf("Clean Job: %v - Next execution: %v", cfg.RunTime.BgJobs.Entry(cleanId).Job, cfg.RunTime.BgJobs.Entry(cleanId).Next.String()))
	}
	if exportErr != nil {
		logger.Error(fmt.Sprintf("Error when scheduling job %v for exporting", exportId), exportErr)
	} else {
		cfg.RunTime.ExportJobId = int(exportId)
		logger.Info(fmt.Sprintf("Export Job: %v - Next execution: %v", cfg.RunTime.BgJobs.Entry(exportId).Job, cfg.RunTime.BgJobs.Entry(exportId).Next.String()))
	}
	logger.Info("Jobs scheduled")
}

func startServer() {
	logger.Info(fmt.Sprintf("Listening on %v", cfg.RunTime.ListenAddr))
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

func cleanUp() {
	cfg.RunTime.BgJobs.Stop()
	shutdownTime := time.Duration(cfg.Server.GracefulShutdownTime) * time.Second
	ctx, cancel = context.WithTimeout(context.Background(), shutdownTime)
	defer func() {
		logger.Info("Cleaning up")
		logger.Info("Done cleaning up")
		cancel()
	}()
}
