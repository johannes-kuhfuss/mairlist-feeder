package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/repositories"
	"github.com/johannes-kuhfuss/mairlist-feeder/service"
	"github.com/stretchr/testify/assert"
)

var (
	repo      repositories.DefaultFileRepository
	uh        StatsUiHandler
	cfg       config.AppConfig
	router    *gin.Engine
	recorder  *httptest.ResponseRecorder
	calCmsSvc service.DefaultCalCmsService
)

func setupUiTest() func() {
	config.InitConfig("", &cfg)
	repo = repositories.NewFileRepository(&cfg)
	calCmsSvc = service.NewCalCmsService(&cfg, &repo)
	uh = NewStatsUiHandler(&cfg, &repo, nil, nil, nil, &calCmsSvc)
	router = gin.Default()
	router.LoadHTMLGlob("../templates/*.tmpl")
	recorder = httptest.NewRecorder()
	return func() {
		router = nil
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
	router.GET("/actions", uh.ActionPage)
	request := httptest.NewRequest(http.MethodGet, "/actions", nil)

	router.ServeHTTP(recorder, request)
	res := recorder.Result()
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	containsTitle := strings.Contains(string(data), "<title>Actions</title>")

	assert.EqualValues(t, http.StatusOK, res.StatusCode)
	assert.Nil(t, err)
	assert.True(t, containsTitle)
}

func TestValidateHourHourEmptyreturnsNoError(t *testing.T) {
	teardown := setupUiTest()
	defer teardown()
	err := validateHour("")
	assert.Nil(t, err)
}

func TestValidateHourInvalidHourreturnsError(t *testing.T) {
	teardown := setupUiTest()
	defer teardown()
	err := validateHour("A")
	assert.NotNil(t, err)
	assert.EqualValues(t, "could not parse hour", err.Message())
	assert.EqualValues(t, 400, err.StatusCode())
}

func TestValidateHourHourTooSmallreturnsError(t *testing.T) {
	teardown := setupUiTest()
	defer teardown()
	err := validateHour("-1")
	assert.NotNil(t, err)
	assert.EqualValues(t, "hour must be between 00 and 23", err.Message())
	assert.EqualValues(t, 400, err.StatusCode())
}

func TestValidateHourHourTooLargereturnsError(t *testing.T) {
	teardown := setupUiTest()
	defer teardown()
	err := validateHour("50")
	assert.NotNil(t, err)
	assert.EqualValues(t, "hour must be between 00 and 23", err.Message())
	assert.EqualValues(t, 400, err.StatusCode())
}

func TestValidateHourValidHourreturnsNoError(t *testing.T) {
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
	router.POST("/actions", uh.ExecAction)
	request := httptest.NewRequest(http.MethodPost, "/actions", nil)

	router.ServeHTTP(recorder, request)
	res := recorder.Result()
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)

	assert.EqualValues(t, http.StatusBadRequest, res.StatusCode)
	assert.EqualValues(t, "{\"message\":\"unknown action\",\"statuscode\":400,\"causes\":null}", string(data))
}

func TestActionExecWrongActionReturnsError(t *testing.T) {
	teardown := setupUiTest()
	defer teardown()
	router.POST("/actions", uh.ExecAction)
	form := url.Values{}
	form.Add("action", "unknown")
	request := httptest.NewRequest(http.MethodPost, "/actions", strings.NewReader(form.Encode()))
	request.Header.Set("Content-type", "application/x-www-form-urlencoded")

	router.ServeHTTP(recorder, request)
	res := recorder.Result()
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)

	assert.EqualValues(t, http.StatusBadRequest, res.StatusCode)
	assert.EqualValues(t, "{\"message\":\"unknown action\",\"statuscode\":400,\"causes\":null}", string(data))
}

func TestActionExecInvalidHourReturnsError(t *testing.T) {
	teardown := setupUiTest()
	defer teardown()
	router.POST("/actions", uh.ExecAction)
	form := url.Values{}
	form.Add("action", "crawl")
	form.Add("hour", "44")
	request := httptest.NewRequest(http.MethodPost, "/actions", strings.NewReader(form.Encode()))
	request.Header.Set("Content-type", "application/x-www-form-urlencoded")

	router.ServeHTTP(recorder, request)
	res := recorder.Result()
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)

	assert.EqualValues(t, http.StatusBadRequest, res.StatusCode)
	assert.EqualValues(t, "{\"message\":\"hour must be between 00 and 23\",\"statuscode\":400,\"causes\":null}", string(data))
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
