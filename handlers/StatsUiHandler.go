package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/dto"
	"github.com/johannes-kuhfuss/mairlist-feeder/repositories"
)

type StatsUiHandler struct {
	Cfg  *config.AppConfig
	Repo *repositories.DefaultFileRepository
}

func NewStatsUiHandler(cfg *config.AppConfig, repo *repositories.DefaultFileRepository) StatsUiHandler {
	return StatsUiHandler{
		Cfg:  cfg,
		Repo: repo,
	}
}

func (uh *StatsUiHandler) StatusPage(c *gin.Context) {
	configData := dto.GetConfig(uh.Cfg)
	c.HTML(http.StatusOK, "status.page.tmpl", gin.H{
		"configdata": configData,
	})
}

func (uh *StatsUiHandler) FileListPage(c *gin.Context) {
	files := dto.GetFiles(uh.Repo)
	c.HTML(http.StatusOK, "filelist.page.tmpl", gin.H{
		"files": files,
	})
}

func (uh *StatsUiHandler) ActionPage(c *gin.Context) {
	c.HTML(http.StatusOK, "actions.page.tmpl", gin.H{
		"data": nil,
	})
}

func (uh *StatsUiHandler) AboutPage(c *gin.Context) {
	c.HTML(http.StatusOK, "about.page.tmpl", nil)
}
