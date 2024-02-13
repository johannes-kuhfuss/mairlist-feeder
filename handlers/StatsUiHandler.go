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
}

func NewStatsUiHandler(cfg *config.AppConfig, repo *repositories.DefaultFileRepository, crs *service.DefaultCrawlService, exs *service.DefaultExportService, cls *service.DefaultCleanService) StatsUiHandler {
	return StatsUiHandler{
		Cfg:       cfg,
		Repo:      repo,
		CrawlSvc:  crs,
		ExportSvc: exs,
		CleanSvc:  cls,
	}
}

func (uh *StatsUiHandler) StatusPage(c *gin.Context) {
	configData := dto.GetConfig(uh.Cfg)
	c.HTML(http.StatusOK, "status.page.tmpl", gin.H{
		"title":      "Status",
		"configdata": configData,
	})
}

func (uh *StatsUiHandler) FileListPage(c *gin.Context) {
	files := dto.GetFiles(uh.Repo)
	c.HTML(http.StatusOK, "filelist.page.tmpl", gin.H{
		"title": "File List",
		"files": files,
	})
}

func (uh *StatsUiHandler) ActionPage(c *gin.Context) {
	c.HTML(http.StatusOK, "actions.page.tmpl", gin.H{
		"title": "Actions",
		"data":  nil,
	})
}

func (uh *StatsUiHandler) AboutPage(c *gin.Context) {
	c.HTML(http.StatusOK, "about.page.tmpl", gin.H{
		"title": "About",
		"data":  nil,
	})
}

func (uh *StatsUiHandler) ExecAction(c *gin.Context) {
	action := c.PostForm("action")
	hour := c.PostForm("hour")
	err := validateAction(action)
	if err != nil {
		logger.Error("Error: ", err)
		c.JSON(err.StatusCode(), err)
		return
	}
	err = validateHour(hour)
	if err != nil {
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
	case "csvexport":
		uh.ExportSvc.ExportToCsv()
	case "importfromdisk":
		uh.Repo.LoadFromDisk(uh.Cfg.Misc.FileSaveFile)
	case "exporttodisk":
		uh.Repo.SaveToDisk(uh.Cfg.Misc.FileSaveFile)
	case "clean":
		uh.CleanSvc.Clean()
	}
	c.JSON(http.StatusOK, nil)
}

func validateAction(action string) api_error.ApiErr {
	actions := []string{"crawl", "export", "clean", "csvexport", "importfromdisk", "exporttodisk"}
	exists := misc.SliceContainsString(actions, action)
	if exists {
		return nil
	} else {
		return api_error.NewBadRequestError("unknown action")
	}
}

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
