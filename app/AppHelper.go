// package app ties together all bits and pieces to start the program
package app

import (
	"fmt"
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

func ExportDayEventsRun() {
	logger.Info("Exporting the day's event's state...")
	file, err := exportDayEvents()
	if err != nil {
		logger.Error("Error exporting day's event's state", err)
	}
	logger.Info(fmt.Sprintf("Exported day's event's state into file %v", file))
}

func exportDayEvents() (fileName string, e error) {
	u := url.URL{}
	if cfg.Server.UseTls {
		u.Scheme = "https"
	} else {
		u.Scheme = "http"
	}
	u.Host = cfg.RunTime.ListenAddr
	u.Path = "/events"
	resp, err := http.Get(u.String())
	if err != nil {
		logger.Error("Error while trying to save day's events", err)
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Error while trying to save day's events", err)
		return "", err
	}
	exportFileName := "events_" + time.Now().Format("2006-01-02__15-04-05") + ".html"
	writePath := path.Join(cfg.Export.ExportFolder, exportFileName)
	absWritePath, err := filepath.Abs(writePath)
	if err != nil {
		logger.Error("error creating event export file path", err)
		return "", err
	}
	if !strings.HasPrefix(absWritePath, cfg.Export.ExportFolder) {
		return "", err
	}
	file, err := os.Create(absWritePath)
	if err != nil {
		logger.Error("error creating event export file", err)
		return "", err
	}
	defer file.Close()
	if _, err := file.Write(body); err != nil {
		logger.Error("error writing event export file", err)
		return "", err
	}
	return absWritePath, nil
}
