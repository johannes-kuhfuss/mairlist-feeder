// package handlers sets up the handlers for the Web UI
package handlers

import (
	"errors"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/johannes-kuhfuss/mairlist-feeder/appstate"
	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/domain"
	"github.com/johannes-kuhfuss/mairlist-feeder/dto"
	"github.com/johannes-kuhfuss/mairlist-feeder/helper"
	"github.com/johannes-kuhfuss/mairlist-feeder/repositories"
	"github.com/johannes-kuhfuss/mairlist-feeder/service"
	"github.com/johannes-kuhfuss/services_utils/api_error"
	"github.com/johannes-kuhfuss/services_utils/logger"
)

type StatsUiHandler struct {
	Cfg       *config.AppConfig
	State     *appstate.AppState
	Repo      repositories.FileRepository
	CrawlSvc  service.Crawler
	ExportSvc uiExporter
	CleanSvc  service.Cleaner
	CalCmsSvc uiCalCmsService
}

type uiExporter interface {
	ExportAllHours() error
	ExportForHour(string) error
}

type uiCalCmsService interface {
	GetTodayEvents() ([]dto.Event, error)
	GetYesterdaysEvents() []dto.Event
}

type actionResponse struct {
	Status  string `json:"status"`
	Action  string `json:"action"`
	Message string `json:"message"`
}

// NewStatsUiHandler creates a new web UI handler and injects its dependencies
func NewStatsUiHandler(cfg *config.AppConfig, repo *repositories.DefaultFileRepository, crs *service.DefaultCrawlService, exs *service.DefaultExportService, cls *service.DefaultCleanService, csv *service.DefaultCalCmsService) StatsUiHandler {
	return NewStatsUiHandlerWithState(cfg, appstate.New(), repo, crs, exs, cls, csv)
}

func NewStatsUiHandlerWithState(cfg *config.AppConfig, state *appstate.AppState, repo repositories.FileRepository, crs service.Crawler, exs uiExporter, cls service.Cleaner, csv uiCalCmsService) StatsUiHandler {
	return StatsUiHandler{
		Cfg:       cfg,
		State:     state,
		Repo:      repo,
		CrawlSvc:  crs,
		ExportSvc: exs,
		CleanSvc:  cls,
		CalCmsSvc: csv,
	}
}

// StatusPage is the handler for the status page
func (uh *StatsUiHandler) StatusPage(c *gin.Context) {
	configData := dto.GetConfig(uh.Cfg, uh.State)
	c.HTML(http.StatusOK, "status.page.tmpl", gin.H{
		"title":      "Status",
		"configdata": configData,
	})
}

// FileListPage is the handler for the file list page
func (uh *StatsUiHandler) FileListPage(c *gin.Context) {
	filterDate, filterDay, err := uh.selectedFilterDate(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	files := dto.GetFilesForDate(uh.Repo, uh.Cfg.CalCms.CmsUrl, filterDate)
	c.HTML(http.StatusOK, "filelist.page.tmpl", gin.H{
		"title":       "File List",
		"files":       files,
		"filterDay":   filterDay,
		"filterRoute": "/filelist",
	})
}

// EventListPage is the handler for the event list page
func (uh *StatsUiHandler) EventListPage(c *gin.Context) {
	filterDate, filterDay, dateErr := uh.selectedFilterDate(c)
	if dateErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": dateErr.Error()})
		return
	}
	events, err := uh.CalCmsSvc.GetTodayEvents()
	if err != nil {
		logger.Error("Error getting today's events", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	events = filterEventsByDate(events, filterDate)
	c.HTML(http.StatusOK, "eventlist.page.tmpl", gin.H{
		"title":       "Event List",
		"events":      events,
		"filterDay":   filterDay,
		"filterRoute": "/events",
		"showFilter":  true,
	})
}

// YesterdaysEvents is the handler for the yesterday's event list page
func (uh *StatsUiHandler) YesterdaysEvents(c *gin.Context) {
	events := uh.CalCmsSvc.GetYesterdaysEvents()
	c.HTML(http.StatusOK, "eventlist.page.tmpl", gin.H{
		"title":      "Yesterday's Event List",
		"events":     events,
		"showFilter": false,
	})
}

func (uh *StatsUiHandler) selectedFilterDate(c *gin.Context) (time.Time, string, error) {
	selectedDay := c.DefaultQuery("day", "today")
	switch selectedDay {
	case "today":
		return helper.DateForFolder(uh.Cfg.Misc.TestCrawl, uh.Cfg.Misc.TestDate, 0), selectedDay, nil
	case "tomorrow":
		return helper.DateForFolder(uh.Cfg.Misc.TestCrawl, uh.Cfg.Misc.TestDate, 1), selectedDay, nil
	default:
		return time.Time{}, "", errors.New("day must be today or tomorrow")
	}
}

func filterEventsByDate(events []dto.Event, filterDate time.Time) []dto.Event {
	filterValue := domain.FormatFolderDate(filterDate)
	filteredEvents := make([]dto.Event, 0, len(events))
	for _, event := range events {
		if event.StartDate == filterValue {
			filteredEvents = append(filteredEvents, event)
		}
	}
	return filteredEvents
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
	note := c.PostForm("note")
	logger.Infof("Execute Action %s with note %v", action, note)
	hour := c.PostForm("hour")
	if err := validateAction(action); err != nil {
		logger.Error("Error validating action", err)
		c.JSON(err.StatusCode(), err)
		return
	}
	if err := validateHour(hour); err != nil {
		logger.Error("Error validating hour", err)
		c.JSON(err.StatusCode(), err)
		return
	}
	switch action {
	case "crawl":
		if err := uh.CrawlSvc.Crawl(); err != nil {
			logger.Error("Error executing crawl action", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
		uh.resetCrawl()
		c.JSON(http.StatusOK, actionResponse{Status: "ok", Action: action, Message: "Crawl completed."})
	case "export":
		if hour == "" {
			if err := uh.ExportSvc.ExportAllHours(); err != nil {
				logger.Error("Error executing export action", err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
				return
			}
			c.JSON(http.StatusOK, actionResponse{Status: "ok", Action: action, Message: "Export completed for all hours."})
		} else {
			if err := uh.ExportSvc.ExportForHour(hour); err != nil {
				logger.Error("Error executing export action", err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
				return
			}
			c.JSON(http.StatusOK, actionResponse{Status: "ok", Action: action, Message: "Export completed for hour " + hour + "."})
		}
	case "exporttodisk":
		if err := uh.Repo.SaveToDisk(uh.Cfg.Misc.FileSaveFile); err != nil {
			logger.Error("Error saving repository to disk", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusOK, actionResponse{Status: "ok", Action: action, Message: "File list saved to disk."})
	case "clean":
		if err := uh.CleanSvc.Clean(); err != nil {
			logger.Error("Error executing clean action", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusOK, actionResponse{Status: "ok", Action: action, Message: "Clean-up completed."})
	}
}

// validateAction filters the actions tring and only allows valid actions
func validateAction(action string) api_error.ApiErr {
	actions := []string{"crawl", "export", "clean", "exporttodisk"}
	if slices.Contains(actions, action) {
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

func (uh *StatsUiHandler) resetCrawl() {
	uh.State.Runtime.BgJobs.Remove(uh.State.Runtime.CrawlJobId)
	crawlCycle := "@every " + strconv.Itoa(uh.Cfg.Crawl.CrawlCycleMin) + "m"
	crawlId, crawlErr := uh.State.Runtime.BgJobs.AddFunc(crawlCycle, func() {
		if err := uh.CrawlSvc.Crawl(); err != nil {
			logger.Error("Error running scheduled crawl", err)
		}
	})
	if crawlErr != nil {
		logger.Errorf("Error when scheduling job %v for crawling. %v", crawlId, crawlErr)
	} else {
		uh.State.Runtime.CrawlJobId = crawlId
	}
}
