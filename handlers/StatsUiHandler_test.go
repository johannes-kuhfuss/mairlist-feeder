package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/html"
)

var (
	uh       StatsUiHandler
	cfg      config.AppConfig
	router   *gin.Engine
	recorder *httptest.ResponseRecorder
)

func setupUiTest(t *testing.T) func() {
	uh = NewStatsUiHandler(&cfg)
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
	request, _ := http.NewRequest(http.MethodGet, "/", nil)

	router.ServeHTTP(recorder, request)

	_, parseErr := html.Parse(io.Reader(recorder.Body))
	assert.EqualValues(t, http.StatusOK, recorder.Code)
	assert.Nil(t, parseErr)
}

func Test_AboutPage_Returns_About(t *testing.T) {
	teardown := setupUiTest(t)
	defer teardown()
	router.GET("/about", uh.AboutPage)
	request, _ := http.NewRequest(http.MethodGet, "/about", nil)

	router.ServeHTTP(recorder, request)

	_, parseErr := html.Parse(io.Reader(recorder.Body))
	assert.EqualValues(t, http.StatusOK, recorder.Code)
	assert.Nil(t, parseErr)
}

func Test_SwitchPage_Returns_Switch(t *testing.T) {
	teardown := setupUiTest(t)
	defer teardown()
	router.GET("/switch", uh.AboutPage)
	request, _ := http.NewRequest(http.MethodGet, "/switch", nil)

	router.ServeHTTP(recorder, request)

	_, parseErr := html.Parse(io.Reader(recorder.Body))
	assert.EqualValues(t, http.StatusOK, recorder.Code)
	assert.Nil(t, parseErr)
}
