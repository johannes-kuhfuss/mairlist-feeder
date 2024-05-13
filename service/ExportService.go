package service

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/domain"
	"github.com/johannes-kuhfuss/mairlist-feeder/helper"
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
	httpExTr       http.Transport
	httpExClient   http.Client
)

func InitHttpExClient() {
	httpExTr = http.Transport{
		DisableKeepAlives:  false,
		DisableCompression: false,
		MaxIdleConns:       0,
		IdleConnTimeout:    0,
	}
	httpExClient = http.Client{
		Transport: &httpExTr,
		Timeout:   5 * time.Second,
	}
}

func NewExportService(cfg *config.AppConfig, repo *repositories.DefaultFileRepository) DefaultExportService {
	fileExportList.Files = make(map[string]domain.FileInfo)
	InitHttpExClient()
	return DefaultExportService{
		Cfg:  cfg,
		Repo: repo,
	}
}

func (s DefaultExportService) Export() {
	nextHour := getNextHour()
	s.ExportForHour(nextHour)
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
			s.checkTimeAndLenghth(files)
			exportPath, err := s.ExportToPlayout(hour)
			if s.Cfg.Export.AppendPlaylist && exportPath != "" && err == nil {
				err := s.AppendPlaylist(exportPath)
				if err != nil {
					logger.Error("Error appending playlist", err)
				}
			}
			logger.Info(fmt.Sprintf("Finished exporting for timeslot %v:00 ...", hour))
		} else {
			logger.Info(fmt.Sprintf("No files to export for timeslot %v:00 ...", hour))
		}
		s.Cfg.RunTime.ExportRunning = false
	}
}

func (s DefaultExportService) checkTimeAndLenghth(files *domain.FileList) {
	for _, file := range *files {
		lengthOk, slotLen, info := checkTime(file, s.Cfg.Export.ShortDeltaAllowance, s.Cfg.Export.LongDeltaAllowance)
		logger.Info(fmt.Sprintf("File: %v, ModDate: %v, IsOK: %v, Info: %v", file.Path, file.ModTime, lengthOk, info))
		if lengthOk {
			file.SlotLength = slotLen
			preFile, exists := fileExportList.Files[createIndexFromTime(file.StartTime)]
			if exists {
				if preFile.ModTime.After(file.ModTime) {
					logger.Info(fmt.Sprintf("Existing file %v is newer than file %v. Not updating.", preFile.Path, file.Path))
				} else {
					logger.Info(fmt.Sprintf("Existing file %v is older than file %v. Updating.", preFile.Path, file.Path))
					fileExportList.Files[createIndexFromTime(file.StartTime)] = file
				}
			} else {
				fileExportList.Files[createIndexFromTime(file.StartTime)] = file
			}
		}
	}
}

func getNextHour() string {
	nextHour := (time.Now().Hour()) + 1
	if nextHour == 24 {
		nextHour = 0
	}
	return fmt.Sprintf("%02d", nextHour)
}

func createIndexFromTime(t1 time.Time) string {
	return t1.Format("15:04")
}

func checkTime(fi domain.FileInfo, shortDelta float64, longDelta float64) (lengthOk bool, slot float64, info string) {
	var (
		lengthSlot   float64
		slotDelta    float64
		plannedDur   float64
		durDelta     float64
		plannedAvail bool
		detail       string
		lenStr       string
	)
	roundedDurationMin := math.Round(fi.Duration / 60)
	is30Min := (roundedDurationMin >= 30.0-shortDelta) && (roundedDurationMin <= 30.0+longDelta)
	is45Min := (roundedDurationMin >= 45.0-shortDelta) && (roundedDurationMin <= 45.0+longDelta)
	is60Min := (roundedDurationMin >= 60.0-shortDelta) && (roundedDurationMin <= 60.0+longDelta)
	is90Min := (roundedDurationMin >= 90.0-shortDelta) && (roundedDurationMin <= 90.0+longDelta)
	is120Min := (roundedDurationMin >= 120.0-shortDelta) && (roundedDurationMin <= 120.0+longDelta)
	switch {
	case is30Min:
		lengthSlot = 30.0
		slotDelta = roundedDurationMin - 30.0
	case is45Min:
		lengthSlot = 45.0
		slotDelta = roundedDurationMin - 45.0
	case is60Min:
		lengthSlot = 60.0
		slotDelta = roundedDurationMin - 60.0
	case is90Min:
		lengthSlot = 90.0
		slotDelta = roundedDurationMin - 90.0
	case is120Min:
		lengthSlot = 120.0
		slotDelta = roundedDurationMin - 120.0
	default:
		lengthSlot = 0.0
		slotDelta = 0.0
	}
	if !fi.EndTime.IsZero() {
		plannedDur = fi.EndTime.Sub(fi.StartTime).Minutes()
		durDelta = roundedDurationMin - plannedDur
		plannedAvail = true
	} else {
		plannedAvail = false
	}
	lOk := is30Min || is45Min || is60Min || is90Min || is120Min
	if lengthSlot > 0.0 {
		lenStr = strconv.Itoa(int(math.Round(lengthSlot))) + "min"

	} else {
		lenStr = "N/A"
	}

	if plannedAvail {
		detail = fmt.Sprintf("Rounded actual duration: %v min, Slot: %v, Delta to slot: %v, planned duration: %v, delta to planned duration: %v",
			roundedDurationMin, lenStr, slotDelta, plannedDur, durDelta)
	} else {
		detail = fmt.Sprintf("Rounded actual duration: %v min, Slot: %v, Delta to slot: %v, no planned duration data available",
			roundedDurationMin, lenStr, slotDelta)
	}
	return lOk, lengthSlot, detail
}

func (s DefaultExportService) ExportToCsv() {
	var (
		startTimeSlot string
		endTimeSlot   string
	)
	size := s.Repo.Size()
	if size > 0 {
		logger.Info(fmt.Sprintf("Exporting %v elements to CSV", size))
		exportPath := path.Join(s.Cfg.Export.ExportFolder, "filescan_export.csv")
		logFile, err := os.OpenFile(exportPath, os.O_APPEND|os.O_CREATE, 0644)
		dataWriter := bufio.NewWriter(logFile)
		if err != nil {
			logger.Error("Error writing csv file: ", err)
		} else {
			files := s.Repo.GetAll()
			if files != nil {
				_, _ = dataWriter.WriteString("Index;StartTime;EndTime;Path;RuleMatched;Length\n")
				for idx, file := range *files {
					if file.StartTime.IsZero() {
						startTimeSlot = "N/A"
					} else {
						startTimeSlot = file.StartTime.Format("15:04")
					}
					if file.EndTime.IsZero() {
						endTimeSlot = "N/A"
					} else {
						endTimeSlot = file.EndTime.Format("15:04")
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
		logger.Info("Done exporting elements to CSV")
	} else {
		logger.Info("No elements to export to CSV")
	}
}

func (s DefaultExportService) ExportToPlayout(hour string) (exportedFile string, err error) {
	// write export list ot mAirlist-compatible file
	// Documentation: https://wiki.mairlist.com/reference:text_playlist_import_format_specification
	// Tab separated
	// Column layout:
	/// 1 - start time = HH:MM
	/// 2 - timing = H (hard fixed time), N (normal)
	/// 3 - line type = F (file), I (database item)
	/// 4 - Line data = full path file name, database Id
	/// 5 - Optional values = omitted here
	var (
		totalLength float64
		startTime   time.Time
	)

	size := len(fileExportList.Files)
	if size > 0 {
		logger.Info(fmt.Sprintf("Exporting %v elements to mAirList for slot %v:00", size, hour))
		exportPath := s.setExportPath(hour)
		exportFile, err := os.OpenFile(exportPath, os.O_CREATE, 0644)
		dataWriter := bufio.NewWriter(exportFile)
		if err != nil {
			logger.Error("Error when creating playlist file for mAirlist: ", err)
			return "", err
		} else {
			s.writeStartComment(dataWriter)
			for time, file := range fileExportList.Files {
				startTime = setStartTime(startTime, time)
				totalLength = totalLength + file.SlotLength
				listTime := time + ":00"
				if s.Cfg.Export.PrependJingle {
					line := fmt.Sprintf("%v\tH\tI\t%v\n", listTime, s.Cfg.Export.JingleIds[rand.Intn(len(s.Cfg.Export.JingleIds))])
					s.writeLine(dataWriter, line)
					line = fmt.Sprintf("\tN\tF\t%v\n", file.Path)
					err := s.writeLine(dataWriter, line)
					if err == nil {
						delete(fileExportList.Files, createIndexFromTime(file.StartTime))
					}
				} else {
					line := fmt.Sprintf("%v\tH\tF\t%v\n", listTime, file.Path)
					err := s.writeLine(dataWriter, line)
					if err == nil {
						delete(fileExportList.Files, createIndexFromTime(file.StartTime))
					}
				}
			}
			if s.Cfg.Export.TerminateAfterDuration {
				s.WriteStopper(dataWriter, startTime, totalLength)
			}
			s.writeEndComment(dataWriter)
			dataWriter.Flush()
			defer exportFile.Close()
			return exportPath, nil
		}
	} else {
		logger.Info(fmt.Sprintf("No elements to export for slot %v:00.", hour))
		return "", nil
	}
}

func setStartTime(startTime time.Time, hour string) time.Time {
	sh, _ := time.Parse("15:04", hour)
	if startTime.IsZero() {
		return sh
	} else {
		if startTime.After(sh) {
			return sh
		} else {
			return startTime
		}
	}
}

func (s DefaultExportService) setExportPath(hour string) string {
	var exportFileName string
	if s.Cfg.Misc.TestCrawl {
		exportFileName = "Test_" + hour + ".tpi"
	} else {
		exportDate := helper.GetTodayFolder(s.Cfg.Misc.TestCrawl, s.Cfg.Misc.TestDate)
		exportFileName = strings.ReplaceAll(exportDate, "/", "-") + "-" + hour + ".tpi"
	}
	s.Cfg.RunTime.LastExportFileName = exportFileName
	s.Cfg.RunTime.LastExportDate = time.Now()
	return path.Join(s.Cfg.Export.ExportFolder, exportFileName)
}

func (s DefaultExportService) writeStartComment(w *bufio.Writer) {
	line := fmt.Sprintf("\t\tR\tPlaylist auto-generated by mAirList Feeder at %v\n", time.Now().Format("2006-01-02 15:04:05"))
	s.writeLine(w, line)
}

func (s DefaultExportService) WriteStopper(w *bufio.Writer, startTime time.Time, tlen float64) {
	dur := time.Duration(tlen * float64(time.Minute))
	eh := startTime.Add(dur)
	eh = eh.Add(-1 * time.Second)
	line := fmt.Sprintf("%v\tH\tI\t%v\n", eh.Format("15:04:05"), s.Cfg.Export.SilenceId)
	s.writeLine(w, line)
}

func (s DefaultExportService) writeEndComment(w *bufio.Writer) {
	line := "\t\tR\tEnd of auto-generated playlist\n"
	s.writeLine(w, line)
}

func (s DefaultExportService) writeLine(w *bufio.Writer, line string) error {
	_, err := w.WriteString(line)
	if err != nil {
		logger.Error("Error when writing playlist entry: ", err)
		return err
	}
	return nil
}

func (s DefaultExportService) AppendPlaylist(fileName string) error {
	// POST to http://<server>:9300/execute
	// Basic Auth
	// Body is Form URL encoded
	// command = PLAYLIST 1 APPEND <filename>
	req, err := s.buildHttpRequest(fileName)
	if err != nil {
		return err
	}
	resp, err := httpExClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode == 404 {
		err := errors.New("url not found")
		return err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == 200 && string(b) == "\"ok\"" {
		return nil
	}
	return errors.New(string(b))
}

func (s DefaultExportService) buildHttpRequest(fileName string) (*http.Request, error) {
	if s.Cfg.Export.MairListUrl == "" {
		return nil, errors.New("url cannot be empty")
	}
	mairListUrl, err := url.Parse(s.Cfg.Export.MairListUrl)
	if err != nil {
		return nil, err
	}
	mairListUrl.Path = path.Join(mairListUrl.Path, "/execute")
	cmd := url.Values{}
	cmd.Set("command", fmt.Sprintf("PLAYLIST 1 APPEND %v", fileName))
	req, _ := http.NewRequest("POST", mairListUrl.String(), strings.NewReader(cmd.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(s.Cfg.Export.MairListUser, s.Cfg.Export.MairListPassword)
	return req, nil
}
