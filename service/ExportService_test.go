package service

import (
	"io"
	"testing"

	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/repositories"
	"github.com/stretchr/testify/assert"
)

var (
	cfg           config.AppConfig
	fileRepo      repositories.DefaultFileRepository
	exportService DefaultExportService
)

func setupTest(t *testing.T) func() {
	config.InitConfig(config.EnvFile, &cfg)
	fileRepo = repositories.NewFileRepository(&cfg)
	exportService = NewExportService(&cfg, &fileRepo)
	return func() {
	}
}

func Test_buildHttpRequest_EmptyUrl_ReturnsError(t *testing.T) {
	tearDown := setupTest(t)
	defer tearDown()
	cfg.Export.MairListUrl = ""

	req, err := exportService.buildHttpRequest("test")

	assert.Nil(t, req)
	assert.NotNil(t, err)
	assert.EqualValues(t, "url cannot be empty", err.Error())
}

func Test_buildHttpRequest_WithUrl_ReturnsRequest(t *testing.T) {
	tearDown := setupTest(t)
	defer tearDown()
	cfg.Export.MairListUrl = "http://localhost:9300/"
	cfg.Export.MairListUser = "test"
	cfg.Export.MairListPassword = "test"

	req, err := exportService.buildHttpRequest("test")

	assert.NotNil(t, req)
	assert.Nil(t, err)
	assert.EqualValues(t, "http://localhost:9300/execute", req.URL.String())
	assert.EqualValues(t, "application/x-www-form-urlencoded", req.Header.Get("Content-Type"))
	assert.EqualValues(t, "Basic dGVzdDp0ZXN0", req.Header.Get("Authorization"))
	b, _ := io.ReadAll(req.Body)
	assert.EqualValues(t, "command=PLAYLIST+1+APPEND+test", string(b))
}
