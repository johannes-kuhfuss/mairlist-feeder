// package app ties together all bits and pieces to start the program
package app

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/johannes-kuhfuss/services_utils/logger"
)

// ExportDayDataRun runs the export job
func ExportDayDataRun() {
	file, err := exportState(eventUrl, "events")
	if err != nil {
		logger.Error("Error exporting day's event state", err)
	} else {
		logger.Infof("Exported day's event state into file %v", file)
	}
	file, err = exportState(fileUrl, "filelist")
	if err != nil {
		logger.Error("Error exporting day's file list state", err)
	} else {
		logger.Infof("Exported day's file list state into file %v", file)
	}
}

// exportState exports an HTML file containing the event view or the file view of the day
// This represents the status for the day, so you can retroactively check for which event files were present
func exportState(urlPath string, filePrefix string) (fileName string, e error) {
	u := url.URL{}
	if cfg.Server.UseTls {
		u.Scheme = "https"
	} else {
		u.Scheme = "http"
	}
	u.Host = cfg.RunTime.ListenAddr
	u.Path = urlPath
	resp, err := http.Get(u.String())
	if err != nil {
		logger.Error("Error while trying to save day's status", err)
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Error while trying to save day's status", err)
		return "", err
	}
	exportFileName := filePrefix + "_" + time.Now().Format("2006-01-02__15-04-05") + ".html"
	writePath := path.Join(cfg.Export.ExportFolder, exportFileName)
	absWritePath, err := filepath.Abs(writePath)
	if err != nil {
		logger.Error("error creating export file path", err)
		return "", err
	}
	if !strings.HasPrefix(absWritePath, cfg.Export.ExportFolder) {
		return "", err
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
