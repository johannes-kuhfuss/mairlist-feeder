// package app ties together all bits and pieces to start the program
package app

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/johannes-kuhfuss/mairlist-feeder/helper"
	"github.com/johannes-kuhfuss/services_utils/logger"
)

var exportStateHTTPClient = &http.Client{
	Timeout: 5 * time.Second,
}

// ExportDayDataRun runs the export job
func (a *Application) ExportDayDataRun() {
	a.calCmsService.SaveYesterdaysEvents()
	file, err := a.exportState(eventUrl, "events")
	if err != nil {
		logger.Error("Error exporting day's event state", err)
	} else {
		logger.Infof("Exported day's event state into file %v", file)
	}
	file, err = a.exportState(fileUrl, "filelist")
	if err != nil {
		logger.Error("Error exporting day's file list state", err)
	} else {
		logger.Infof("Exported day's file list state into file %v", file)
	}
}

// exportState exports an HTML file containing the event view or the file view of the day
// This represents the status for the day, so you can retroactively check for which event files were present
func (a *Application) exportState(urlPath, filePrefix string) (fileName string, e error) {
	body, err := a.exportStateBody(urlPath)
	if err != nil {
		return "", err
	}
	exportFileName := filePrefix + "_" + time.Now().Format("2006-01-02") + ".html"
	writePath := filepath.Join(a.cfg.Export.ExportFolder, exportFileName)
	absWritePath, err := filepath.Abs(writePath)
	if err != nil {
		logger.Error("error creating export file path", err)
		return "", err
	}
	if !helper.IsPathWithin(absWritePath, a.cfg.Export.ExportFolder) {
		return "", errors.New("invalid export path")
	}
	file, err := os.Create(absWritePath)
	if err != nil {
		logger.Error("error creating export file", err)
		return "", err
	}
	defer file.Close()
	if _, err := file.Write(body); err != nil {
		logger.Error("error writing export file", err)
		return "", err
	}
	return absWritePath, nil
}

func (a *Application) exportStateBody(urlPath string) ([]byte, error) {
	if a.state != nil && a.state.Runtime.Snapshot().Router != nil {
		router := a.state.Runtime.Snapshot().Router
		req := httptest.NewRequest(http.MethodGet, urlPath, nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			return nil, fmt.Errorf("status export request failed for %s: %s", urlPath, http.StatusText(resp.Code))
		}
		return resp.Body.Bytes(), nil
	}
	u := url.URL{}
	if a.cfg.Server.UseTLS {
		u.Scheme = "https"
	} else {
		u.Scheme = "http"
	}
	u.Host = a.listenAddr()
	u.Path = urlPath
	resp, err := exportStateHTTPClient.Get(u.String())
	if err != nil {
		logger.Error("Error while trying to save day's status", err)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status export request failed for %s: %s", urlPath, resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Error while trying to save day's status", err)
		return nil, err
	}
	return body, nil
}

func (a *Application) listenAddr() string {
	if a.state == nil {
		return ""
	}
	return a.state.Runtime.Snapshot().ListenAddr
}
