package service

import (
	"bufio"
	"fmt"
	"math"
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
	var (
		startTimeSlot string
		endTimeSlot   string
	)
	logFile, err := os.OpenFile("filescan.csv", os.O_APPEND|os.O_CREATE, 0644)
	dataWriter := bufio.NewWriter(logFile)
	if err != nil {
		logger.Error("Error writing log file: ", err)
	} else {
		files := s.Repo.GetAll()
		if files != nil {
			_, _ = dataWriter.WriteString("Index;StartTime;EndTime;Path;RuleMatched;Length\n")
			for idx, file := range *files {
				if file.StartTime == "" {
					startTimeSlot = "N/A"
				} else {
					startTimeSlot = file.StartTime
				}
				if file.EndTime == "" {
					endTimeSlot = "N/A"
				} else {
					endTimeSlot = file.EndTime
				}
				infoString := fmt.Sprintf("%04d;%v;%v;%v;%v;%v\n", idx, startTimeSlot, endTimeSlot, file.Path, file.RuleMatched, math.Round(file.Duration))
				_, _ = dataWriter.WriteString(infoString)
			}
		} else {
			logger.Info("No files found to export.")
		}
	}
	dataWriter.Flush()
	logFile.Close()
	logger.Info("Finished exporting")
}
