package app

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHealthzReturnsOk(t *testing.T) {
	gin.SetMode(gin.TestMode)
	a := Application{}
	a.cfg.Crawl.RootFolder = t.TempDir()
	a.cfg.Export.ExportFolder = t.TempDir()
	router := gin.New()
	router.GET("/healthz", a.healthz)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	assert.EqualValues(t, http.StatusOK, recorder.Code)
	assert.EqualValues(t, "{\"status\":\"ok\"}", recorder.Body.String())
}

func TestHealthzMissingRootFolderReturnsUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	a := Application{}
	a.cfg.Crawl.RootFolder = filepath.Join(t.TempDir(), "missing")
	a.cfg.Export.ExportFolder = t.TempDir()
	router := gin.New()
	router.GET("/healthz", a.healthz)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	assert.EqualValues(t, http.StatusServiceUnavailable, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "root folder is not accessible")
}

func TestHealthzMissingExportFolderReturnsUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	a := Application{}
	a.cfg.Crawl.RootFolder = t.TempDir()
	a.cfg.Export.ExportFolder = filepath.Join(t.TempDir(), "missing")
	router := gin.New()
	router.GET("/healthz", a.healthz)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	assert.EqualValues(t, http.StatusServiceUnavailable, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "export folder is not accessible")
}
