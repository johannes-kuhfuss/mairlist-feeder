package service

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/domain"
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

var (
	fileExportList domain.SafeFileList
)

func NewExportService(cfg *config.AppConfig, repo *repositories.DefaultFileRepository) DefaultExportService {
	fileExportList.Files = make(map[string]domain.FileInfo)
	return DefaultExportService{
		Cfg:  cfg,
		Repo: repo,
	}
}

func (s DefaultExportService) Export() {
	nextHour := getNextHour()
	logger.Info(fmt.Sprintf("Starting export for timeslot %v:00 ...", nextHour))
	files := s.Repo.GetForHour(nextHour)
	if files != nil {
		for _, file := range *files {
			lengthOk, info := checkTime(file, s.Cfg.Export.ShortDeltaAllowance, s.Cfg.Export.LongDeltaAllowance)
			logger.Info(fmt.Sprintf("File: %v, IsOK: %v, Info: %v", file.Path, lengthOk, info))
			fileExportList.Files[file.StartTime] = file
			/// remove duplicates / determine latest version
			/// Add to export list
		}
	} else {
		logger.Info(fmt.Sprintf("No files to export for timeslot %v:00 ...", nextHour))
	}
	// write export list ot mAirlist-compatible file
	logger.Info(fmt.Sprintf("Finished exporting for timeslot %v:00 ...", nextHour))
}

func getNextHour() string {
	/*
		nextHour := (time.Now().Hour()) + 1
		return fmt.Sprintf("%02d", nextHour)
	*/
	return "20"
}

func checkTime(fi domain.FileInfo, shortDelta float64, longDelta float64) (lengthOk bool, info string) {
	var (
		lengthSlot   string
		slotDelta    float64
		plannedDur   float64
		durDelta     float64
		plannedAvail bool
	)
	roundedDurationMin := math.Round(fi.Duration / 60)
	is30Min := (roundedDurationMin >= 30.0-shortDelta) && (roundedDurationMin <= 30.0+longDelta)
	is45Min := (roundedDurationMin >= 45.0-shortDelta) && (roundedDurationMin <= 45.0+longDelta)
	is60Min := (roundedDurationMin >= 60.0-shortDelta) && (roundedDurationMin <= 60.0+longDelta)
	is90Min := (roundedDurationMin >= 90.0-shortDelta) && (roundedDurationMin <= 90.0+longDelta)
	is120Min := (roundedDurationMin >= 120.0-shortDelta) && (roundedDurationMin <= 120.0+longDelta)
	switch {
	case is30Min:
		lengthSlot = "30min"
		slotDelta = roundedDurationMin - 30.0
	case is45Min:
		lengthSlot = "45min"
		slotDelta = roundedDurationMin - 45.0
	case is60Min:
		lengthSlot = "60min"
		slotDelta = roundedDurationMin - 60.0
	case is90Min:
		lengthSlot = "90min"
		slotDelta = roundedDurationMin - 90.0
	case is120Min:
		lengthSlot = "120min"
		slotDelta = roundedDurationMin - 120.0
	default:
		lengthSlot = "N/A"
		slotDelta = 0.0
	}
	if fi.EndTime != "" {
		start, _ := time.Parse("15:04", fi.StartTime)
		end, _ := time.Parse("15:04", fi.EndTime)
		plannedDur = end.Sub(start).Minutes()
		durDelta = roundedDurationMin - plannedDur
		plannedAvail = true
	} else {
		plannedAvail = false
	}
	lOk := is30Min || is45Min || is60Min || is90Min || is120Min
	detail := fmt.Sprintf("Rounded actual duration: %v min, Slot: %v, Delta to slot: %v, planned duration data available: %v, planned duration: %v, delta to planned duration: %v",
		roundedDurationMin, lengthSlot, slotDelta, plannedAvail, plannedDur, durDelta)
	return lOk, detail
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
