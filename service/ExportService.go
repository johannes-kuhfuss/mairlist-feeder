package service

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"path"
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
	if s.Cfg.Misc.TestExport {
		s.ExportAllHours()
	} else {
		nextHour := getNextHour()
		s.ExportForHour(nextHour)
	}
}

func (s DefaultExportService) ExportAllHours() {
	for hour := 0; hour < 24; hour++ {
		s.ExportForHour(fmt.Sprintf("%02d", hour))
	}
}

func (s DefaultExportService) ExportForHour(hour string) {
	if s.Cfg.RunTime.ExportRunning {
		logger.Warn("Export already running. Not starting another one.")
	} else {
		s.Cfg.RunTime.ExportRunning = true
		files := s.Repo.GetForHour(hour)
		if files != nil {
			logger.Info(fmt.Sprintf("Starting export for timeslot %v:00 ...", hour))
			for _, file := range *files {
				lengthOk, info := checkTime(file, s.Cfg.Export.ShortDeltaAllowance, s.Cfg.Export.LongDeltaAllowance)
				logger.Info(fmt.Sprintf("File: %v, ModDate: %v, IsOK: %v, Info: %v", file.Path, file.ModTime, lengthOk, info))
				if lengthOk {
					preFile, exists := fileExportList.Files[file.StartTime]
					if exists {
						if preFile.ModTime.After(file.ModTime) {
							logger.Info(fmt.Sprintf("Existing file %v is newer than file %v. Not updating.", preFile.Path, file.Path))
						} else {
							logger.Info(fmt.Sprintf("Existing file %v is older than file %v. Updating.", preFile.Path, file.Path))
							fileExportList.Files[file.StartTime] = file
						}
					}
					fileExportList.Files[file.StartTime] = file
				}
			}
			s.ExportToPlayout(hour)
			logger.Info(fmt.Sprintf("Finished exporting for timeslot %v:00 ...", hour))
		} else {
			logger.Info(fmt.Sprintf("No files to export for timeslot %v:00 ...", hour))
		}
		s.Cfg.RunTime.ExportRunning = false
	}
}

func getNextHour() string {
	nextHour := (time.Now().Hour()) + 1
	if nextHour == 24 {
		nextHour = 0
	}
	return fmt.Sprintf("%02d", nextHour)
}

func checkTime(fi domain.FileInfo, shortDelta float64, longDelta float64) (lengthOk bool, info string) {
	var (
		lengthSlot   string
		slotDelta    float64
		plannedDur   float64
		durDelta     float64
		plannedAvail bool
		detail       string
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
	if plannedAvail {
		detail = fmt.Sprintf("Rounded actual duration: %v min, Slot: %v, Delta to slot: %v, planned duration: %v, delta to planned duration: %v",
			roundedDurationMin, lengthSlot, slotDelta, plannedDur, durDelta)
	} else {
		detail = fmt.Sprintf("Rounded actual duration: %v min, Slot: %v, Delta to slot: %v, no planned duration data available",
			roundedDurationMin, lengthSlot, slotDelta)
	}
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

func (s DefaultExportService) ExportToPlayout(hour string) {
	// write export list ot mAirlist-compatible file
	// Documentation: https://wiki.mairlist.com/reference:text_playlist_import_format_specification
	// Tab separated
	// Column layout:
	/// 1 - start time = HH:MM
	/// 2 - timing = H (hard fixed time)
	/// 3 - line type = F (file)
	/// 4 - Line data = full path file name
	/// 5 - Optional values = omitted here
	var (
		exportPath     string
		exportFileName string
	)
	size := len(fileExportList.Files)
	if size > 0 {
		logger.Info(fmt.Sprintf("Exporting %v elements to mAirList for slot %v:00", size, hour))
		if s.Cfg.Misc.TestCrawl {
			exportFileName = "Test_" + hour + ".txt"
		} else {
			year := fmt.Sprintf("%d", time.Now().Year())
			month := fmt.Sprintf("%02d", time.Now().Month())
			day := fmt.Sprintf("%02d", time.Now().Day())
			exportFileName = year + "-" + month + "-" + day + "-" + hour + ".txt"
		}
		s.Cfg.RunTime.LastExportFileName = exportFileName
		s.Cfg.RunTime.LastExportDate = time.Now()
		exportPath = path.Join(s.Cfg.Export.ExportFolder, exportFileName)
		exportFile, err := os.OpenFile(exportPath, os.O_CREATE, 0644)
		dataWriter := bufio.NewWriter(exportFile)
		if err != nil {
			logger.Error("Error when creating playlist file for mAirlist: ", err)
		} else {
			for time, file := range fileExportList.Files {
				expLine := fmt.Sprintf("%v\tH\tF\t%v\n", time, file.Path)
				_, err := dataWriter.WriteString(expLine)
				if err != nil {
					logger.Error("Error when writing playlist entry: ", err)
				} else {
					delete(fileExportList.Files, file.StartTime)
				}
			}
			dataWriter.Flush()
			defer exportFile.Close()
		}
	} else {
		logger.Info(fmt.Sprintf("No elements to export for slot %v:00.", hour))
	}

}
