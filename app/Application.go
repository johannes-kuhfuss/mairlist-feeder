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
	"github.com/robfig/cron"

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
	bgJobs         *cron.Cron
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
	if cfg.Misc.LoadFromDisk {
		fileRepo.LoadFromDisk(cfg.Misc.FileSaveFile)
	} else {
		crawlService.Crawl()
	}
	if cfg.Misc.Test {
		exportService.Export()
	}

	go startServer()

	<-appEnd
	cleanUp()

	if srvErr := server.Shutdown(ctx); err != nil {
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
	gin.DefaultWriter = logger.GetLogger()
	router := gin.New()
	router.Use(gin.Logger())
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
	statsUiHandler = handlers.NewStatsUiHandler(&cfg)
	fileRepo = repositories.NewFileRepository(&cfg)
	crawlService = service.NewCrawlService(&cfg, &fileRepo)
	cleanService = service.NewCleanService(&cfg, &fileRepo)
	exportService = service.NewExportService(&cfg, &fileRepo)
}

func mapUrls() {
	cfg.RunTime.Router.GET("/", statsUiHandler.StatusPage)
	cfg.RunTime.Router.GET("/about", statsUiHandler.AboutPage)
}

func RegisterForOsSignals() {
	appEnd = make(chan os.Signal, 1)
	signal.Notify(appEnd, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
}

func scheduleBgJobs() {
	logger.Info("Scheduling jobs...")
	crawlCycle := "@every " + strconv.Itoa(cfg.Crawl.CrawlCycleMin) + "m"
	bgJobs = cron.New()
	// Crawl every x minutes
	bgJobs.AddFunc(crawlCycle, crawlService.Crawl)
	// Clean 02:03 local time
	bgJobs.AddFunc("0 3 2 * * *", cleanService.Clean)
	// Export every hour, 10 minutes to the hour
	bgJobs.AddFunc("0 50 * * * *", exportService.Export)
	bgJobs.Start()
	for _, job := range bgJobs.Entries() {
		logger.Info(fmt.Sprintf("Job: %v - Next execution: %v", job.Job, job.Next.String()))
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
	bgJobs.Stop()
	shutdownTime := time.Duration(cfg.Server.GracefulShutdownTime) * time.Second
	ctx, cancel = context.WithTimeout(context.Background(), shutdownTime)
	defer func() {
		logger.Info("Cleaning up")
		logger.Info("Done cleaning up")
		cancel()
	}()
}
