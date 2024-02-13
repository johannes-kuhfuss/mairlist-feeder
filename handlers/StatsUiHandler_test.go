package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/repositories"
	"github.com/stretchr/testify/assert"
)

var (
	repo     repositories.DefaultFileRepository
	uh       StatsUiHandler
	cfg      config.AppConfig
	router   *gin.Engine
	recorder *httptest.ResponseRecorder
)

func setupUiTest(t *testing.T) func() {
	repo = repositories.NewFileRepository(&cfg)
	uh = NewStatsUiHandler(&cfg, &repo, nil, nil, nil)
	router = gin.Default()
	router.LoadHTMLGlob("../templates/*.tmpl")
	recorder = httptest.NewRecorder()
	return func() {
		router = nil
	}
}

func Test_StatusPage_Returns_Status(t *testing.T) {
	teardown := setupUiTest(t)
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

func Test_AboutPage_Returns_About(t *testing.T) {
	teardown := setupUiTest(t)
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

func Test_FileListPage_Returns_FileListPage(t *testing.T) {
	teardown := setupUiTest(t)
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

func Test_ActionPage_Returns_Action(t *testing.T) {
	teardown := setupUiTest(t)
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

func Test_validateHour_HourEmpty_returnsNoError(t *testing.T) {
	teardown := setupUiTest(t)
	defer teardown()
	err := validateHour("")
	assert.Nil(t, err)
}

func Test_validateHour_InvalidHour_returnsError(t *testing.T) {
	teardown := setupUiTest(t)
	defer teardown()
	err := validateHour("A")
	assert.NotNil(t, err)
	assert.EqualValues(t, "could not parse hour", err.Message())
	assert.EqualValues(t, 400, err.StatusCode())
}

func Test_validateHour_HourTooSmall_returnsError(t *testing.T) {
	teardown := setupUiTest(t)
	defer teardown()
	err := validateHour("-1")
	assert.NotNil(t, err)
	assert.EqualValues(t, "hour must be between 00 and 23", err.Message())
	assert.EqualValues(t, 400, err.StatusCode())
}

func Test_validateHour_HourTooLarge_returnsError(t *testing.T) {
	teardown := setupUiTest(t)
	defer teardown()
	err := validateHour("50")
	assert.NotNil(t, err)
	assert.EqualValues(t, "hour must be between 00 and 23", err.Message())
	assert.EqualValues(t, 400, err.StatusCode())
}

func Test_validateHour_ValidHour_returnsNoError(t *testing.T) {
	teardown := setupUiTest(t)
	defer teardown()
	err := validateHour("2")
	assert.Nil(t, err)
}

func Test_validateAction_UnkownAction_ReturnsError(t *testing.T) {
	teardown := setupUiTest(t)
	defer teardown()
	err := validateAction("unknown")
	assert.NotNil(t, err)
	assert.EqualValues(t, "unknown action", err.Message())
	assert.EqualValues(t, 400, err.StatusCode())
}

func Test_validateAction_CorrectAction_ReturnsNoError(t *testing.T) {
	teardown := setupUiTest(t)
	defer teardown()
	actions := []string{"crawl", "export", "clean", "csvexport", "importfromdisk", "exporttodisk"}
	for _, action := range actions {
		err := validateAction(action)
		assert.Nil(t, err)
	}
}
