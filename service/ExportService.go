package service

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"time"

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
	nextHour := getNextHour()
	//nextHour = "21"
	logger.Info(fmt.Sprintf("Starting export for timeslot %v:00 ...", nextHour))
	files := s.Repo.GetForHour(nextHour)
	if files != nil {
		for _, file := range *files {
			/// remove duplicates / determine latest version
			/// perform sanity check on duration
			//// Absolute duration: 30min+/-2min, 45min+/-2min, 60min+/-2min, 75min+/-2min, 90min+/-2min, 120min+/-2min
			//// Compare to end time if available
			/// Add to export list
			logger.Info(file.Path)
		}
	} else {
		logger.Info(fmt.Sprintf("No files to export for timeslot %v:00 ...", nextHour))
	}
	// write export list ot mAirlist-compatible file
	logger.Info(fmt.Sprintf("Finished exporting for timeslot %v:00 ...", nextHour))
}

func getNextHour() string {
	nextHour := (time.Now().Hour()) + 1
	return fmt.Sprintf("%02d", nextHour)
}

func (s DefaultExportService) exportToCsv() {
	var (
		startTimeSlot string
		endTimeSlot   string
	)
	logFile, err := os.OpenFile("filescan.csv", os.O_APPEND|os.O_CREATE, 0644)
	dataWriter := bufio.NewWriter(logFile)
	if err != nil {
		logger.Error("Error writing csv file: ", err)
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
			logger.Info("No file entries found to export.")
		}
	}
	dataWriter.Flush()
	logFile.Close()
}
