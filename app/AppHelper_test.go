package app

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/stretchr/testify/assert"
)

func setupHelperTest() {
	config.InitConfig(config.EnvFile, &cfg)
	cfg.RunTime.ListenAddr = fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
}

func TestExportDayEventsNoConfigReturnsError(t *testing.T) {
	file, err := exportDayEvents()
	assert.NotNil(t, err)
	assert.EqualValues(t, "", file)
	assert.EqualValues(t, "Get \"http:///events\": http: no Host in request URL", err.Error())
}

func TestExportDayEventsGetErrorReturnsError(t *testing.T) {
	setupHelperTest()
	file, err := exportDayEvents()
	assert.NotNil(t, err)
	assert.EqualValues(t, "", file)
	assert.EqualValues(t, "Get \"http://:8080/events\": dial tcp :8080: connectex: No connection could be made because the target machine actively refused it.", err.Error())
}

func TestExportDayEventsNoErrorReturnsFileName(t *testing.T) {
	setupHelperTest()
	htmlResp := "<!DOCTYPE html><html><body></body></html>"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(htmlResp))
	}))
	defer srv.Close()
	u, _ := url.Parse(srv.URL)
	cfg.RunTime.ListenAddr = u.Hostname() + ":" + u.Port()
	fileName, err := exportDayEvents()
	_, noFile := os.Stat(fileName)
	assert.Nil(t, err)
	assert.Nil(t, noFile)
	time.Sleep(1 * time.Second)
	os.Remove(fileName)
}
