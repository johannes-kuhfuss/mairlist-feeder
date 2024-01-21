package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/dto"
)

type StatsUiHandler struct {
	Cfg *config.AppConfig
}

func NewStatsUiHandler(cfg *config.AppConfig) StatsUiHandler {
	return StatsUiHandler{
		Cfg: cfg,
	}
}

func (uh *StatsUiHandler) StatusPage(c *gin.Context) {
	configData := dto.GetConfig(uh.Cfg)
	c.HTML(http.StatusOK, "status.page.tmpl", gin.H{
		"configdata": configData,
	})
}

func (uh *StatsUiHandler) FileListPage(c *gin.Context) {
	fileData := dto.GetFiles(uh.Cfg)
	c.HTML(http.StatusOK, "filelist.page.tmpl", gin.H{
		"filedata": fileData,
	})
}

func (uh *StatsUiHandler) AboutPage(c *gin.Context) {
	c.HTML(http.StatusOK, "about.page.tmpl", nil)
}
