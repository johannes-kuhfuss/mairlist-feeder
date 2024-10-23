// package service implements the services and their business logic that provide the main part of the program
package service

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
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

var (
	exmu sync.Mutex
)

// The export service handles the export of information to mAirList
type DefaultExportService struct {
	Cfg  *config.AppConfig
	Repo *repositories.DefaultFileRepository
}

var (
	fileExportList domain.SafeFileList
	httpExTr       http.Transport
	httpExClient   http.Client
)

// InitHttpExClient sets the defaukt values for the http client used to interact with mAirlist
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

// NewExportService creates a new export service and injects its dependencies
func NewExportService(cfg *config.AppConfig, repo *repositories.DefaultFileRepository) DefaultExportService {
	fileExportList.Files = make(map[string]domain.FileInfo)
	InitHttpExClient()
	return DefaultExportService{
		Cfg:  cfg,
		Repo: repo,
	}
}

// isNotMonday is a helper function to determine if today is not Monday
func isNotMonday() bool {
	return int(time.Now().Weekday()) != 1
}

// Export orchestrates the export of data to mAirList
func (s DefaultExportService) Export() {
	s.Cfg.RunTime.LastExportRunDate = time.Now()
	nextHour := getNextHour()
	// LimitTime is a stop-gap measure until the human-side of the new TK is squared away. Will be removed once mairlist-feeder goes into full production
	if s.Cfg.Export.LimitTime {
		// 23:00, but not on Mondays and 00:00, 01:00
		if (nextHour == "23" && isNotMonday()) || (nextHour == "00") || (nextHour == "01") {
			s.ExportForHour(nextHour)
		}
	} else {
		s.ExportForHour(nextHour)
	}

}

// ExportAllHours exports a playlist for all hours of the day
func (s DefaultExportService) ExportAllHours() {
	for hour := 0; hour < 24; hour++ {
		s.ExportForHour(fmt.Sprintf("%02d", hour))
	}
}

// ExportForHour exports a playlist for a given hour
// Loads playlist into mAirList via API, if enabled
func (s DefaultExportService) ExportForHour(hour string) {
	exmu.Lock()
	defer exmu.Unlock()
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

// checkTimeAndLenghth determines suitability of files for playout, based on their length
// Also resolves conflicts if there are multiple matching files for the same time
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

// getNextHour is a helper function that returns the next hour
func getNextHour() string {
	nextHour := (time.Now().Hour()) + 1
	if nextHour == 24 {
		nextHour = 0
	}
	return fmt.Sprintf("%02d", nextHour)
}

// createIndexFromTime is a helper function that returns the time in HH:MM format
func createIndexFromTime(t1 time.Time) string {
	return t1.Format("15:04")
}

// checkTime is a helper function that compares available time information such as start time, end time, etc.
// It classifies the entry into a slot length and calculates differences between the actual file length and the presumed slot length
// Finally it makes a determination whether the file's length is OK to play the file
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

// ExportToPlayout writes a ".tpi" playlist to disk for a given hour
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
		line        string
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
				if file.FromCalCMS && !file.EndTime.IsZero() {
					plannedDur := file.EndTime.Sub(file.StartTime).Minutes()
					totalLength = totalLength + plannedDur
				} else {
					totalLength = totalLength + file.SlotLength
				}
				listTime := time + ":00"
				switch file.FileType {
				case "Stream":
					line = fmt.Sprintf("%v\tH\tI\t%v\n", listTime, file.StreamId)
				default:
					line = fmt.Sprintf("%v\tH\tF\t%v\n", listTime, file.Path)
				}
				err := s.writeLine(dataWriter, line)
				if err == nil {
					delete(fileExportList.Files, createIndexFromTime(file.StartTime))
				}
			}
			if s.Cfg.Export.TerminateAfterDuration {
				s.WriteStopper(dataWriter, startTime, totalLength)
			}
			s.writeEndComment(dataWriter)
			dataWriter.Flush()
			defer exportFile.Close()
			s.Cfg.RunTime.LastExportFileName = exportPath
			s.Cfg.RunTime.LastExportedFileDate = time.Now()
			return exportPath, nil
		}
	} else {
		logger.Info(fmt.Sprintf("No elements to export for slot %v:00.", hour))
		return "", nil
	}
}

// setStartTime is a helper function that determines the correct start time value for a playlist element
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

// setExportPath is a helper function creating the export path for the ".tpi" playlist file
func (s DefaultExportService) setExportPath(hour string) string {
	var exportFileName string
	if s.Cfg.Misc.TestCrawl {
		exportFileName = "Test_" + hour + ".tpi"
	} else {
		exportDate := helper.GetTodayFolder(s.Cfg.Misc.TestCrawl, s.Cfg.Misc.TestDate)
		exportFileName = strings.ReplaceAll(exportDate, "/", "-") + "-" + hour + ".tpi"
	}
	return path.Join(s.Cfg.Export.ExportFolder, exportFileName)
}

// writeStartComment is a helper function creating the ".tpi" file's start comment
func (s DefaultExportService) writeStartComment(w *bufio.Writer) {
	line := fmt.Sprintf("\t\tR\tPlaylist auto-generated by mAirList Feeder at %v\n", time.Now().Format("2006-01-02 15:04:05"))
	s.writeLine(w, line)
}

// WriteStopper is a helper function adding an end element to the ".tpi" file
func (s DefaultExportService) WriteStopper(w *bufio.Writer, startTime time.Time, tlen float64) {
	dur := time.Duration(tlen * float64(time.Minute))
	eh := startTime.Add(dur)
	line := fmt.Sprintf("%v\tH\tD\tEnd of block\n", eh.Format("15:04:05"))
	s.writeLine(w, line)
}

// writeEndComment is a helper function creating the ".tpi" file's end comment
func (s DefaultExportService) writeEndComment(w *bufio.Writer) {
	line := "\t\tR\tEnd of auto-generated playlist\n"
	s.writeLine(w, line)
}

// writeLine is a helper function that writes agiven line to file
func (s DefaultExportService) writeLine(w *bufio.Writer, line string) error {
	_, err := w.WriteString(line)
	if err != nil {
		logger.Error("Error when writing playlist entry: ", err)
		return err
	}
	return nil
}

// AppendPlaylist appends the playlist written to the ".tpi" file to the current playlist in mAirList using the API
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
	if resp.StatusCode == 200 && string(b) == "ok" {
		return nil
	}
	return errors.New(string(b))
}

// buildHttpRequest is a helper function constructing the mAirList API request to append the playlist
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
