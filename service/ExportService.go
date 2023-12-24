package service

import (
	"bufio"
	"fmt"
	"os"

	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/repositories"
	"github.com/johannes-kuhfuss/services_utils/logger"
)

type ExportService interface {
	Export()
}

type DefaultExportService struct {
	Cfg  *config.AppConfig
	Repo *repositories.DefaultFileRepository
}

func NewExportService(cfg *config.AppConfig, repo *repositories.DefaultFileRepository) DefaultExportService {
	return DefaultExportService{
		Cfg:  cfg,
		Repo: repo,
	}
}

func (s DefaultExportService) Export() {
	logger.Info("Starting export...")
	var timeSlot string
	logFile, err := os.OpenFile("filescan.txt", os.O_APPEND|os.O_CREATE, 0644)
	dataWriter := bufio.NewWriter(logFile)
	if err != nil {
		logger.Error("Error writing log file: ", err)
	} else {
		files := s.Repo.GetAll()
		for idx, file := range *files {

			if file.StartTime == "" {
				timeSlot = "N/A"
			} else {
				timeSlot = file.StartTime
			}
			infoString := fmt.Sprintf("Index: %04d - Time slot: %v - Path: %v\n", idx, timeSlot, file.Path)
			_, _ = dataWriter.WriteString(infoString)
		}
	}
	dataWriter.Flush()
	logFile.Close()
	logger.Info("Finished exporting")
}
