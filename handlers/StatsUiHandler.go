// package handlers sets up the handlers for the Web UI
package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/dto"
	"github.com/johannes-kuhfuss/mairlist-feeder/repositories"
	"github.com/johannes-kuhfuss/mairlist-feeder/service"
	"github.com/johannes-kuhfuss/services_utils/api_error"
	"github.com/johannes-kuhfuss/services_utils/logger"
	"github.com/johannes-kuhfuss/services_utils/misc"
)

type StatsUiHandler struct {
	Cfg       *config.AppConfig
	Repo      *repositories.DefaultFileRepository
	CrawlSvc  *service.DefaultCrawlService
	ExportSvc *service.DefaultExportService
	CleanSvc  *service.DefaultCleanService
	CalCmsSvc *service.DefaultCalCmsService
}

// NewStatsUiHandler creates a new web UI handler and injects its dependencies
func NewStatsUiHandler(cfg *config.AppConfig, repo *repositories.DefaultFileRepository, crs *service.DefaultCrawlService, exs *service.DefaultExportService, cls *service.DefaultCleanService, csv *service.DefaultCalCmsService) StatsUiHandler {
	return StatsUiHandler{
		Cfg:       cfg,
		Repo:      repo,
		CrawlSvc:  crs,
		ExportSvc: exs,
		CleanSvc:  cls,
		CalCmsSvc: csv,
	}
}

// StatusPage is the handler for the status page
func (uh *StatsUiHandler) StatusPage(c *gin.Context) {
	configData := dto.GetConfig(uh.Cfg)
	c.HTML(http.StatusOK, "status.page.tmpl", gin.H{
		"title":      "Status",
		"configdata": configData,
	})
}

// FileListPage is the handler for the file list page
func (uh *StatsUiHandler) FileListPage(c *gin.Context) {
	files := dto.GetFiles(uh.Repo, uh.Cfg.CalCms.CmsUrl)
	c.HTML(http.StatusOK, "filelist.page.tmpl", gin.H{
		"title": "File List",
		"files": files,
	})
}

// EventListPage is the handler for the event list page
func (uh *StatsUiHandler) EventListPage(c *gin.Context) {
	events, _ := uh.CalCmsSvc.GetEvents()
	c.HTML(http.StatusOK, "eventlist.page.tmpl", gin.H{
		"title":  "Event List",
		"events": events,
	})
}

// ActionPage is the handler for the page where the user can invoke actions
func (uh *StatsUiHandler) ActionPage(c *gin.Context) {
	c.HTML(http.StatusOK, "actions.page.tmpl", gin.H{
		"title": "Actions",
		"data":  nil,
	})
}

// LogsPage is the handler for the page displaying log messages
func (uh *StatsUiHandler) LogsPage(c *gin.Context) {
	logs := logger.GetLogList()
	c.HTML(http.StatusOK, "logs.page.tmpl", gin.H{
		"title": "Logs",
		"logs":  logs,
	})
}

// AboutPage is the handler for the page displaying a short description of the program and its license
func (uh *StatsUiHandler) AboutPage(c *gin.Context) {
	c.HTML(http.StatusOK, "about.page.tmpl", gin.H{
		"title": "About",
		"data":  nil,
	})
}

// ExecAction is the handler invoked when the user excecutes an action
func (uh *StatsUiHandler) ExecAction(c *gin.Context) {
	action := c.PostForm("action")
	hour := c.PostForm("hour")
	if err := validateAction(action); err != nil {
		logger.Error("Error: ", err)
		c.JSON(err.StatusCode(), err)
		return
	}
	if err := validateHour(hour); err != nil {
		logger.Error("Error: ", err)
		c.JSON(err.StatusCode(), err)
		return
	}
	switch action {
	case "crawl":
		uh.CrawlSvc.Crawl()
	case "export":
		if hour == "" {
			uh.ExportSvc.ExportAllHours()
		} else {
			uh.ExportSvc.ExportForHour(hour)
		}
	case "exporttodisk":
		uh.Repo.SaveToDisk(uh.Cfg.Misc.FileSaveFile)
	case "clean":
		uh.CleanSvc.Clean()
	}
	c.JSON(http.StatusOK, nil)
}

// validateAction filters the actions tring and only allows valid actions
func validateAction(action string) api_error.ApiErr {
	actions := []string{"crawl", "export", "clean", "exporttodisk"}
	if exists := misc.SliceContainsString(actions, action); exists {
		return nil
	} else {
		return api_error.NewBadRequestError("unknown action")
	}
}

// validateHour validates the hour input by the user and only allows valid hours
func validateHour(hour string) api_error.ApiErr {
	if hour == "" {
		return nil
	}
	h, err := strconv.Atoi(hour)
	if err != nil {
		return api_error.NewBadRequestError("could not parse hour")
	}
	if (h < 0) || (h > 23) {
		return api_error.NewBadRequestError("hour must be between 00 and 23")
	}
	return nil
}
