package handlers

import (
	"encoding/json"
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
	"github.com/johannes-kuhfuss/mairlist-feeder/appstate"
	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/domain"
	metrics "github.com/johannes-kuhfuss/mairlist-feeder/metrics"
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
	state     *appstate.AppState
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
	state = appstate.New()
	metrics.InitMetrics(state, registry)
	repo = repositories.NewFileRepository(&cfg)
	calCmsSvc = service.NewCalCmsServiceWithState(&cfg, state, &repo)
	crawlSvc = service.NewCrawlServiceWithState(&cfg, state, &repo, &calCmsSvc)
	exportSvc = service.NewExportServiceWithState(&cfg, state, &repo)
	cleanSvc = service.NewCleanServiceWithState(&cfg, state, &repo)
	state.Runtime.BgJobs = cron.New()
	crawlJobID, _ := state.Runtime.BgJobs.AddFunc("@every 10m", func() {
		_ = crawlSvc.Crawl()
	})
	state.Runtime.CrawlJobID = crawlJobID
	uh = NewStatsUiHandlerWithState(&cfg, state, &repo, &crawlSvc, &exportSvc, &cleanSvc, &calCmsSvc)
	router = gin.Default()
	router.LoadHTMLGlob("../templates/*.tmpl")
	recorder = httptest.NewRecorder()
	return func() {
		uh.Close()
		state.Runtime.BgJobs.Stop()
		router = nil
		metrics.UnregisterMetrics(state, registry)
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

func TestFileListPageDefaultsToTodayFilter(t *testing.T) {
	teardown := setupUiTest()
	defer teardown()
	cfg.Misc.TestCrawl = true
	cfg.Misc.TestDate = "2024/09/17"
	repo.Store(domain.FileInfo{Path: "today-file", FolderDate: domain.MustParseFolderDate("2024-09-17")})
	repo.Store(domain.FileInfo{Path: "tomorrow-file", FolderDate: domain.MustParseFolderDate("2024-09-18")})
	router.GET("/filelist", uh.FileListPage)
	request := httptest.NewRequest(http.MethodGet, "/filelist", nil)

	router.ServeHTTP(recorder, request)
	res := recorder.Result()
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	body := string(data)

	assert.EqualValues(t, http.StatusOK, res.StatusCode)
	assert.Nil(t, err)
	assert.Contains(t, body, `option value="today" selected`)
	assert.Contains(t, body, "today-file")
	assert.NotContains(t, body, "tomorrow-file")
}

func TestFileListPageUsesDateQueryFilter(t *testing.T) {
	teardown := setupUiTest()
	defer teardown()
	cfg.Misc.TestCrawl = true
	cfg.Misc.TestDate = "2024/09/17"
	repo.Store(domain.FileInfo{Path: "today-file", FolderDate: domain.MustParseFolderDate("2024-09-17")})
	repo.Store(domain.FileInfo{Path: "tomorrow-file", FolderDate: domain.MustParseFolderDate("2024-09-18")})
	router.GET("/filelist", uh.FileListPage)
	request := httptest.NewRequest(http.MethodGet, "/filelist?day=tomorrow", nil)

	router.ServeHTTP(recorder, request)
	res := recorder.Result()
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	body := string(data)

	assert.EqualValues(t, http.StatusOK, res.StatusCode)
	assert.Nil(t, err)
	assert.Contains(t, body, `option value="tomorrow" selected`)
	assert.NotContains(t, body, "today-file")
	assert.Contains(t, body, "tomorrow-file")
}

func TestFileListPageRejectsUnknownDayFilter(t *testing.T) {
	teardown := setupUiTest()
	defer teardown()
	router.GET("/filelist", uh.FileListPage)
	request := httptest.NewRequest(http.MethodGet, "/filelist?day=next-week", nil)

	router.ServeHTTP(recorder, request)
	res := recorder.Result()
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)

	assert.EqualValues(t, http.StatusBadRequest, res.StatusCode)
	assert.Nil(t, err)
	assert.EqualValues(t, "{\"message\":\"day must be today or tomorrow\"}", string(data))
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
	assert.Contains(t, body, `await pollAction(data.status_url, statusField)`)
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

func waitForActionJob(t *testing.T, data []byte) actionJob {
	t.Helper()
	var submitted actionJob
	assert.NoError(t, json.Unmarshal(data, &submitted))
	var completed actionJob
	assert.Eventually(t, func() bool {
		job, ok := uh.jobs.get(submitted.ID)
		if ok {
			completed = job
		}
		return ok && (job.Status == "succeeded" || job.Status == "failed")
	}, 3*time.Second, 10*time.Millisecond)
	return completed
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
	cfg.Crawl.RootFolder = t.TempDir()
	router.POST(actionUrl, uh.ExecAction)
	form := url.Values{}
	form.Add("action", "crawl")

	data, statusCode := runRequest(form)

	assert.EqualValues(t, http.StatusAccepted, statusCode)
	job := waitForActionJob(t, data)
	assert.Equal(t, "succeeded", job.Status)
	assert.Equal(t, "Crawl completed.", job.Message)
	assert.True(t, state.Runtime.BgJobs.Entry(state.Runtime.CrawlJobID).Valid())
}

func TestActionExecCleanReturnsOk(t *testing.T) {
	teardown := setupUiTest()
	defer teardown()
	router.POST(actionUrl, uh.ExecAction)
	repo.Store(domain.FileInfo{
		Path:       "old-file",
		FolderDate: domain.NormalizeDate(time.Now().AddDate(0, 0, -1)),
	})
	form := url.Values{}
	form.Add("action", "clean")

	data, statusCode := runRequest(form)

	assert.EqualValues(t, http.StatusAccepted, statusCode)
	job := waitForActionJob(t, data)
	assert.Equal(t, "succeeded", job.Status)
	assert.Equal(t, "Clean-up completed.", job.Message)
	assert.EqualValues(t, 1, state.Runtime.FilesCleaned)
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

	assert.EqualValues(t, http.StatusAccepted, statusCode)
	job := waitForActionJob(t, data)
	assert.Equal(t, "succeeded", job.Status)
	assert.Equal(t, "Export completed for hour 13.", job.Message)
}

func TestActionStatusReturnsCompletedJob(t *testing.T) {
	teardown := setupUiTest()
	defer teardown()
	router.POST(actionUrl, uh.ExecAction)
	router.GET(actionUrl+"/:id", uh.ActionStatus)
	form := url.Values{"action": {"export"}, "hour": {"13"}}

	data, statusCode := runRequest(form)
	job := waitForActionJob(t, data)
	statusRecorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, job.StatusURL, nil)
	router.ServeHTTP(statusRecorder, request)

	assert.Equal(t, http.StatusAccepted, statusCode)
	assert.Equal(t, http.StatusOK, statusRecorder.Code)
	var response actionJob
	assert.NoError(t, json.Unmarshal(statusRecorder.Body.Bytes(), &response))
	assert.Equal(t, "succeeded", response.Status)
	assert.Equal(t, "Export completed for hour 13.", response.Message)
}

func TestActionStatusReturnsNotFound(t *testing.T) {
	teardown := setupUiTest()
	defer teardown()
	router.GET(actionUrl+"/:id", uh.ActionStatus)
	request := httptest.NewRequest(http.MethodGet, actionUrl+"/missing", nil)

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestActionExecExportToDiskReturnsOk(t *testing.T) {
	teardown := setupUiTest()
	defer teardown()
	cfg.Misc.FileSaveFile = filepath.Join(t.TempDir(), "files.dta")
	router.POST(actionUrl, uh.ExecAction)
	form := url.Values{}
	form.Add("action", "exporttodisk")

	data, statusCode := runRequest(form)

	assert.EqualValues(t, http.StatusAccepted, statusCode)
	job := waitForActionJob(t, data)
	assert.Equal(t, "succeeded", job.Status)
	assert.Equal(t, "File list saved to disk.", job.Message)
	_, statErr := os.Stat(cfg.Misc.FileSaveFile)
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

func TestEventsPageDefaultsToTodayFilter(t *testing.T) {
	teardown := setupUiTest()
	defer teardown()
	cfg.Misc.TestCrawl = true
	cfg.Misc.TestDate = "2024/09/17"
	seedUiEvents(t)
	router.GET("/events", uh.EventListPage)
	request := httptest.NewRequest(http.MethodGet, "/events", nil)

	router.ServeHTTP(recorder, request)
	res := recorder.Result()
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	body := string(data)

	assert.EqualValues(t, http.StatusOK, res.StatusCode)
	assert.Nil(t, err)
	assert.Contains(t, body, `option value="today" selected`)
	assert.Contains(t, body, "Today Event")
	assert.NotContains(t, body, "Tomorrow Event")
}

func TestEventsPageUsesDateQueryFilter(t *testing.T) {
	teardown := setupUiTest()
	defer teardown()
	cfg.Misc.TestCrawl = true
	cfg.Misc.TestDate = "2024/09/17"
	seedUiEvents(t)
	router.GET("/events", uh.EventListPage)
	request := httptest.NewRequest(http.MethodGet, "/events?day=tomorrow", nil)

	router.ServeHTTP(recorder, request)
	res := recorder.Result()
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	body := string(data)

	assert.EqualValues(t, http.StatusOK, res.StatusCode)
	assert.Nil(t, err)
	assert.Contains(t, body, `option value="tomorrow" selected`)
	assert.NotContains(t, body, "Today Event")
	assert.Contains(t, body, "Tomorrow Event")
}

func TestEventsPageRejectsUnknownDayFilter(t *testing.T) {
	teardown := setupUiTest()
	defer teardown()
	router.GET("/events", uh.EventListPage)
	request := httptest.NewRequest(http.MethodGet, "/events?day=next-week", nil)

	router.ServeHTTP(recorder, request)
	res := recorder.Result()
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)

	assert.EqualValues(t, http.StatusBadRequest, res.StatusCode)
	assert.Nil(t, err)
	assert.EqualValues(t, "{\"message\":\"day must be today or tomorrow\"}", string(data))
}

func seedUiEvents(t *testing.T) {
	t.Helper()
	calCmsData := domain.CalCmsPgmData{
		Events: []domain.CalCmsEvent{
			{
				EventID:       1,
				FullTitle:     "Today Event",
				StartDate:     "2024-09-17",
				StartTime:     "1100",
				EndTime:       "1200",
				Duration:      "01:00:00",
				StartDatetime: "2024-09-17T11:00:00",
				EndDatetime:   "2024-09-17T12:00:00",
			},
			{
				EventID:       2,
				FullTitle:     "Tomorrow Event",
				StartDate:     "2024-09-18",
				StartTime:     "1100",
				EndTime:       "1200",
				Duration:      "01:00:00",
				StartDatetime: "2024-09-18T11:00:00",
				EndDatetime:   "2024-09-18T12:00:00",
			},
		},
	}
	respData, err := json.Marshal(calCmsData)
	assert.Nil(t, err)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Add("Content-Type", "application/json")
		w.Write(respData)
	}))
	t.Cleanup(srv.Close)
	cfg.CalCms.CmsUrl = srv.URL
	cfg.CalCms.QueryCalCms = true
	_, err = calCmsSvc.RefreshTodayEvents()
	assert.Nil(t, err)
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
