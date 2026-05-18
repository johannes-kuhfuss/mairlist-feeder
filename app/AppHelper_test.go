package app

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/stretchr/testify/assert"
)

func setupHelperTest() {
	testApp = Application{}
	config.InitConfig(config.EnvFile, &testApp.cfg)
	testApp.cfg.RunTime.ListenAddr = fmt.Sprintf("%s:%s", testApp.cfg.Server.Host, testApp.cfg.Server.Port)
}

var testApp Application

func TestExportDayEventsNoConfigReturnsError(t *testing.T) {
	testApp = Application{}
	file, err := testApp.exportState(eventUrl, "events")
	assert.NotNil(t, err)
	assert.EqualValues(t, "", file)
	assert.EqualValues(t, "Get \"http:///events\": http: no Host in request URL", err.Error())
}

func TestExportDayEventsGetErrorReturnsError(t *testing.T) {
	setupHelperTest()
	file, err := testApp.exportState(eventUrl, "events")
	assert.NotNil(t, err)
	assert.EqualValues(t, "", file)
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
	testApp.cfg.RunTime.ListenAddr = u.Hostname() + ":" + u.Port()
	fileName, err := testApp.exportState(eventUrl, "events")
	_, noFile := os.Stat(fileName)
	assert.Nil(t, err)
	assert.Nil(t, noFile)
	time.Sleep(1 * time.Second)
	os.Remove(fileName)
}

func TestIsPathWithinRejectsSiblingDirectory(t *testing.T) {
	root := filepath.Join(t.TempDir(), "export")
	candidate := root + "2"

	assert.False(t, isPathWithin(candidate, root))
}
