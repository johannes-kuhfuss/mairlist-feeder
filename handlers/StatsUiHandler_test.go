package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	metrics "github.com/johannes-kuhfuss/mairlist-feeder/Metrics"
	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/domain"
	"github.com/johannes-kuhfuss/mairlist-feeder/repositories"
	"github.com/johannes-kuhfuss/mairlist-feeder/service"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/robfig/cron/v3"
	"github.com/stretchr/testify/assert"
)

var (
	repo      repositories.DefaultFileRepository
	uh        StatsUiHandler
	cfg       config.AppConfig
	router    *gin.Engine
	recorder  *httptest.ResponseRecorder
	calCmsSvc service.DefaultCalCmsService
	crawlSvc  service.DefaultCrawlService
	exportSvc service.DefaultExportService
	cleanSvc  service.DefaultCleanService
)

const (
	actionUrl = "/actions"
)

func setupUiTest() func() {
	registry := prometheus.NewRegistry()
	config.InitConfig("", &cfg)
	metrics.InitMetrics(&cfg, registry)
	repo = repositories.NewFileRepository(&cfg)
	calCmsSvc = service.NewCalCmsService(&cfg, &repo)
	crawlSvc = service.NewCrawlService(&cfg, &repo, &calCmsSvc)
	exportSvc = service.NewExportService(&cfg, &repo)
	cleanSvc = service.NewCleanService(&cfg, &repo)
	cfg.RunTime.BgJobs = cron.New()
	crawlJobId, _ := cfg.RunTime.BgJobs.AddFunc("@every 10m", crawlSvc.Crawl)
	cfg.RunTime.CrawlJobId = crawlJobId
	uh = NewStatsUiHandler(&cfg, &repo, &crawlSvc, &exportSvc, &cleanSvc, &calCmsSvc)
	router = gin.Default()
	router.LoadHTMLGlob("../templates/*.tmpl")
	recorder = httptest.NewRecorder()
	return func() {
		cfg.RunTime.BgJobs.Stop()
		router = nil
		metrics.UnregisterMetrics(&cfg, registry)
	}
}

func TestStatusPageReturnsStatus(t *testing.T) {
	teardown := setupUiTest()
	defer teardown()
	router.GET("/", uh.StatusPage)
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	router.ServeHTTP(recorder, request)
	res := recorder.Result()
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	containsTitle := strings.Contains(string(data), "<title>Status</title>")

	assert.EqualValues(t, http.StatusOK, res.StatusCode)
	assert.Nil(t, err)
	assert.True(t, containsTitle)
}

func TestAboutPageReturnsAbout(t *testing.T) {
	teardown := setupUiTest()
	defer teardown()
	router.GET("/about", uh.AboutPage)
	request := httptest.NewRequest(http.MethodGet, "/about", nil)

	router.ServeHTTP(recorder, request)
	res := recorder.Result()
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	containsTitle := strings.Contains(string(data), "<title>About</title>")

	assert.EqualValues(t, http.StatusOK, res.StatusCode)
	assert.Nil(t, err)
	assert.True(t, containsTitle)
}

func TestFileListPageReturnsFileListPage(t *testing.T) {
	teardown := setupUiTest()
	defer teardown()
	router.GET("/filelist", uh.FileListPage)
	request := httptest.NewRequest(http.MethodGet, "/filelist", nil)

	router.ServeHTTP(recorder, request)
	res := recorder.Result()
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	containsTitle := strings.Contains(string(data), "<title>File List</title>")

	assert.EqualValues(t, http.StatusOK, res.StatusCode)
	assert.Nil(t, err)
	assert.True(t, containsTitle)
}

func TestActionPageReturnsAction(t *testing.T) {
	teardown := setupUiTest()
	defer teardown()
	router.GET(actionUrl, uh.ActionPage)
	request := httptest.NewRequest(http.MethodGet, actionUrl, nil)

	router.ServeHTTP(recorder, request)
	res := recorder.Result()
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	containsTitle := strings.Contains(string(data), "<title>Actions</title>")

	assert.EqualValues(t, http.StatusOK, res.StatusCode)
	assert.Nil(t, err)
	assert.True(t, containsTitle)
}

func TestActionPageContainsFeedbackUi(t *testing.T) {
	teardown := setupUiTest()
	defer teardown()
	router.GET(actionUrl, uh.ActionPage)
	request := httptest.NewRequest(http.MethodGet, actionUrl, nil)

	router.ServeHTTP(recorder, request)
	res := recorder.Result()
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	body := string(data)

	assert.EqualValues(t, http.StatusOK, res.StatusCode)
	assert.Nil(t, err)
	assert.Contains(t, body, `id="status"`)
	assert.Contains(t, body, `role="status"`)
	assert.Contains(t, body, `await fetch("/actions"`)
	assert.Contains(t, body, `alert alert-success`)
	assert.Contains(t, body, `alert alert-danger`)
	assert.Contains(t, body, `data.message || "Action completed."`)
}

func TestValidateHourHourEmptyReturnsNoError(t *testing.T) {
	teardown := setupUiTest()
	defer teardown()
	err := validateHour("")
	assert.Nil(t, err)
}

func TestValidateHourInvalidHourReturnsError(t *testing.T) {
	teardown := setupUiTest()
	defer teardown()
	err := validateHour("A")
	assert.NotNil(t, err)
	assert.EqualValues(t, "could not parse hour", err.Message())
	assert.EqualValues(t, 400, err.StatusCode())
}

func TestValidateHourHourTooSmallReturnsError(t *testing.T) {
	teardown := setupUiTest()
	defer teardown()
	err := validateHour("-1")
	assert.NotNil(t, err)
	assert.EqualValues(t, "hour must be between 00 and 23", err.Message())
	assert.EqualValues(t, 400, err.StatusCode())
}

func TestValidateHourHourTooLargeReturnsError(t *testing.T) {
	teardown := setupUiTest()
	defer teardown()
	err := validateHour("50")
	assert.NotNil(t, err)
	assert.EqualValues(t, "hour must be between 00 and 23", err.Message())
	assert.EqualValues(t, 400, err.StatusCode())
}

func TestValidateHourValidHourReturnsNoError(t *testing.T) {
	teardown := setupUiTest()
	defer teardown()
	err := validateHour("2")
	assert.Nil(t, err)
}

func TestValidateActionUnkownActionReturnsError(t *testing.T) {
	teardown := setupUiTest()
	defer teardown()
	err := validateAction("unknown")
	assert.NotNil(t, err)
	assert.EqualValues(t, "unknown action", err.Message())
	assert.EqualValues(t, 400, err.StatusCode())
}

func TestValidateActionCorrectActionReturnsNoError(t *testing.T) {
	teardown := setupUiTest()
	defer teardown()
	actions := []string{"crawl", "export", "clean", "exporttodisk"}
	for _, action := range actions {
		err := validateAction(action)
		assert.Nil(t, err)
	}
}

func TestActionExecNoDataReturnsError(t *testing.T) {
	teardown := setupUiTest()
	defer teardown()
	router.POST(actionUrl, uh.ExecAction)
	request := httptest.NewRequest(http.MethodPost, actionUrl, nil)

	router.ServeHTTP(recorder, request)
	res := recorder.Result()
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)

	assert.EqualValues(t, http.StatusBadRequest, res.StatusCode)
	assert.EqualValues(t, "{\"message\":\"unknown action\",\"statuscode\":400,\"causes\":null}", string(data))
}

func runRequest(form url.Values) (data []byte, statusCode int) {
	request := httptest.NewRequest(http.MethodPost, actionUrl, strings.NewReader(form.Encode()))
	request.Header.Set("Content-type", "application/x-www-form-urlencoded")
	router.ServeHTTP(recorder, request)
	res := recorder.Result()
	defer res.Body.Close()
	data, _ = io.ReadAll(res.Body)
	return data, res.StatusCode
}

func TestActionExecWrongActionReturnsError(t *testing.T) {
	teardown := setupUiTest()
	defer teardown()
	router.POST(actionUrl, uh.ExecAction)
	form := url.Values{}
	form.Add("action", "unknown")

	data, statusCode := runRequest(form)

	assert.EqualValues(t, http.StatusBadRequest, statusCode)
	assert.EqualValues(t, "{\"message\":\"unknown action\",\"statuscode\":400,\"causes\":null}", string(data))
}

func TestActionExecInvalidHourReturnsError(t *testing.T) {
	teardown := setupUiTest()
	defer teardown()
	router.POST(actionUrl, uh.ExecAction)
	form := url.Values{}
	form.Add("action", "crawl")
	form.Add("hour", "44")

	data, statusCode := runRequest(form)

	assert.EqualValues(t, http.StatusBadRequest, statusCode)
	assert.EqualValues(t, "{\"message\":\"hour must be between 00 and 23\",\"statuscode\":400,\"causes\":null}", string(data))
}

func TestActionExecCrawlReturnsOk(t *testing.T) {
	teardown := setupUiTest()
	defer teardown()
	router.POST(actionUrl, uh.ExecAction)
	form := url.Values{}
	form.Add("action", "crawl")

	data, statusCode := runRequest(form)

	assert.EqualValues(t, http.StatusOK, statusCode)
	assert.EqualValues(t, "{\"status\":\"ok\",\"action\":\"crawl\",\"message\":\"Crawl completed.\"}", string(data))
	assert.True(t, cfg.RunTime.BgJobs.Entry(cfg.RunTime.CrawlJobId).Valid())
}

func TestActionExecCleanReturnsOk(t *testing.T) {
	teardown := setupUiTest()
	defer teardown()
	router.POST(actionUrl, uh.ExecAction)
	repo.Store(domain.FileInfo{
		Path:       "old-file",
		FolderDate: time.Now().AddDate(0, 0, -1).Format("2006-01-02"),
	})
	form := url.Values{}
	form.Add("action", "clean")

	data, statusCode := runRequest(form)

	assert.EqualValues(t, http.StatusOK, statusCode)
	assert.EqualValues(t, "{\"status\":\"ok\",\"action\":\"clean\",\"message\":\"Clean-up completed.\"}", string(data))
	assert.EqualValues(t, 1, cfg.RunTime.FilesCleaned)
	assert.EqualValues(t, 0, repo.Size())
}

func TestActionExecExportReturnsOk(t *testing.T) {
	teardown := setupUiTest()
	defer teardown()
	router.POST(actionUrl, uh.ExecAction)
	form := url.Values{}
	form.Add("action", "export")
	form.Add("hour", "13")

	data, statusCode := runRequest(form)

	assert.EqualValues(t, http.StatusOK, statusCode)
	assert.EqualValues(t, "{\"status\":\"ok\",\"action\":\"export\",\"message\":\"Export completed for hour 13.\"}", string(data))
}

func TestActionExecExportToDiskReturnsOk(t *testing.T) {
	teardown := setupUiTest()
	defer teardown()
	cfg.Misc.FileSaveFile = filepath.Join(t.TempDir(), "files.dta")
	router.POST(actionUrl, uh.ExecAction)
	form := url.Values{}
	form.Add("action", "exporttodisk")

	data, statusCode := runRequest(form)
	_, statErr := os.Stat(cfg.Misc.FileSaveFile)

	assert.EqualValues(t, http.StatusOK, statusCode)
	assert.EqualValues(t, "{\"status\":\"ok\",\"action\":\"exporttodisk\",\"message\":\"File list saved to disk.\"}", string(data))
	assert.Nil(t, statErr)
}

func TestLogsPageReturnsLogs(t *testing.T) {
	teardown := setupUiTest()
	defer teardown()
	router.GET("/logs", uh.LogsPage)
	request := httptest.NewRequest(http.MethodGet, "/logs", nil)

	router.ServeHTTP(recorder, request)
	res := recorder.Result()
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	containsTitle := strings.Contains(string(data), "<title>Logs</title>")

	assert.EqualValues(t, http.StatusOK, res.StatusCode)
	assert.Nil(t, err)
	assert.True(t, containsTitle)
}

func TestEventsPageReturnsEvents(t *testing.T) {
	teardown := setupUiTest()
	defer teardown()
	cfg.CalCms.QueryCalCms = true
	router.GET("/events", uh.EventListPage)
	request := httptest.NewRequest(http.MethodGet, "/events", nil)

	router.ServeHTTP(recorder, request)
	res := recorder.Result()
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	containsTitle := strings.Contains(string(data), "<title>Event List</title>")

	assert.EqualValues(t, http.StatusOK, res.StatusCode)
	assert.Nil(t, err)
	assert.True(t, containsTitle)
}

func TestYesterdayPageReturnsYesterdaysEvents(t *testing.T) {
	teardown := setupUiTest()
	defer teardown()
	cfg.CalCms.QueryCalCms = true
	router.GET("/yesterday", uh.YesterdaysEvents)
	request := httptest.NewRequest(http.MethodGet, "/yesterday", nil)

	router.ServeHTTP(recorder, request)
	res := recorder.Result()
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	containsTitle := strings.Contains(string(data), "<title>Yesterday&#39;s Event List</title>")

	assert.EqualValues(t, http.StatusOK, res.StatusCode)
	assert.Nil(t, err)
	assert.True(t, containsTitle)
}
