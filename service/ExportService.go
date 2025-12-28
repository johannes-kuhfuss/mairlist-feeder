// package service implements the services and their business logic that provide the main part of the program
package service

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/domain"
	"github.com/johannes-kuhfuss/mairlist-feeder/dto"
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

const (
	dateFormat = "2006-01-02 15:04:05 -0700 MST"
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

// InitHttpExClient sets the default values for the http client used to interact with mAirlist
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

// Export orchestrates the export of data to mAirList
func (s DefaultExportService) Export() {
	s.Cfg.RunTime.LastExportRunDate = time.Now()
	nextHour := getNextHour()
	s.ExportForHour(nextHour)
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
	if files := s.Repo.GetByHour(hour, s.Cfg.Export.ExportLiveItems); files != nil {
		logger.Infof("Starting export for timeslot %v:00 ...", hour)
		start := time.Now().UTC()
		sort.Sort(files)
		s.checkTimeAndLenghth(files)
		exportPath, err := s.ExportToPlayout(hour)
		if s.Cfg.Export.AppendPlaylist && exportPath != "" && err == nil {
			req := dto.MairListRequest{
				ReqType:  "appendpl",
				FileName: exportPath,
			}
			err := s.ExecuteMairListRequest(req)
			if err != nil {
				logger.Error("Error appending playlist", err)
			}
		}
		end := time.Now().UTC()
		dur := end.Sub(start)
		logger.Infof("Finished exporting for timeslot %v:00 ... (%v)", hour, dur.String())
	} else {
		logger.Infof("No files to export for timeslot %v:00 ...", hour)
	}
	s.Cfg.RunTime.ExportRunning = false
}

// checkTimeAndLenghth determines suitability of files for playout, based on their length
// Also resolves conflicts if there are multiple matching files for the same time
func (s DefaultExportService) checkTimeAndLenghth(files *domain.FileList) {
	for _, file := range *files {
		lengthOk, slotLen, info := checkTime(file, s.Cfg.Export.ShortDeltaAllowance, s.Cfg.Export.LongDeltaAllowance)
		logger.Infof("File: %v, ModDate: %v, IsOK: %v, Info: %v", file.Path, file.ModTime, lengthOk, info)
		if lengthOk {
			file.SlotLength = slotLen
			preFile, exists := fileExportList.Files[createIndexFromTime(file.StartTime)]
			if exists {
				if preFile.ModTime.After(file.ModTime) {
					logger.Infof("Existing file %v is newer than file %v. Not updating.", preFile.Path, file.Path)
				} else {
					logger.Infof("Existing file %v is older than file %v. Updating.", preFile.Path, file.Path)
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
	isLonger := (roundedDurationMin > 120.0+longDelta)
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
	case isLonger:
		lengthSlot = roundedDurationMin
		slotDelta = 0.0
		logger.Warnf("Detected very long file: %v with length %vmin. Please verify.", fi.Path, roundedDurationMin)
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
	lOk := is30Min || is45Min || is60Min || is90Min || is120Min || isLonger
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
	if size := len(fileExportList.Files); size > 0 {
		logger.Infof("Exporting %v elements to mAirList for slot %v:00", size, hour)
		exportPath, err := s.setExportPath(hour)
		if err != nil {
			logger.Error("Error when setting export path", err)
			return "", err
		}
		if err := s.WritePlaylist(exportPath); err == nil {
			s.Cfg.RunTime.LastExportFileName = exportPath
			s.Cfg.RunTime.LastExportedFileDate = time.Now()
			return exportPath, nil
		}
		return "", err
	}
	logger.Infof("No elements to export for slot %v:00.", hour)
	return "", nil
}

func (s DefaultExportService) WritePlaylist(exportPath string) error {
	var (
		totalLength float64
		startTime   time.Time
		line        string
	)
	exportFile, err := os.OpenFile(exportPath, os.O_CREATE, 0644)
	dataWriter := bufio.NewWriter(exportFile)
	if err != nil {
		logger.Error("Error when creating playlist file for mAirlist", err)
		return err
	}
	defer exportFile.Close()
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
		if err := s.writeLine(dataWriter, line); err == nil {
			delete(fileExportList.Files, createIndexFromTime(file.StartTime))
		}
	}
	if s.Cfg.Export.TerminateAfterDuration {
		s.WriteStopper(dataWriter, startTime, totalLength)
	}
	s.writeEndComment(dataWriter)
	dataWriter.Flush()
	return nil
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
func (s DefaultExportService) setExportPath(hour string) (exportPath string, e error) {
	var exportFileName string
	if s.Cfg.Misc.TestCrawl {
		exportFileName = "Test_" + hour + ".tpi"
	} else {
		exportDate := helper.GetTodayFolder(s.Cfg.Misc.TestCrawl, s.Cfg.Misc.TestDate)
		exportFileName = strings.ReplaceAll(exportDate, "/", "-") + "-" + hour + ".tpi"
	}
	expPath := path.Join(s.Cfg.Export.ExportFolder, exportFileName)
	absExpPath, err := filepath.Abs(expPath)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(absExpPath, s.Cfg.Export.ExportFolder) {
		return "", errors.New("invalid export path")
	}
	return absExpPath, nil
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
		logger.Error("Error when writing playlist entry", err)
		return err
	}
	return nil
}

func (s DefaultExportService) ExecuteMairListRequest(req dto.MairListRequest) error {
	// ReqType = appendpl, getpl
	switch req.ReqType {
	case "appendpl":
		err := s.AppendPlaylist(req.FileName)
		if err != nil {
			return err
		}
	case "getpl":
		err := s.GetPlaylist()
		if err != nil {
			return err
		}
	default:
		return errors.New("not implemented")
	}
	return nil
}

// AppendPlaylist appends the playlist written to the ".tpi" file to the current playlist in mAirList using the API
func (s DefaultExportService) AppendPlaylist(fileName string) error {
	// POST to http://<server>:9300/execute
	// Basic Auth
	// Body is Form URL encoded
	// command = PLAYLIST 1 APPEND <filename>
	var (
		okReply bool
	)
	req, err := s.buildAppendRequest(fileName)
	if err != nil {
		return err
	}
	data, statusCode, err := s.execRequest(req)
	if err != nil {
		return err
	}
	if s.Cfg.Export.MairListVersion >= 6 {
		okReply = string(data) == "\"ok\""
	} else {
		okReply = string(data) == "ok"
	}
	if statusCode == 200 && okReply {
		logger.Infof("Successfully appended playlist %v to mAirList", fileName)
		return nil
	}
	return errors.New(string(data))
}

func (s DefaultExportService) SetMairListCommState(success bool) {
	s.Cfg.RunTime.Mu.Lock()
	defer s.Cfg.RunTime.Mu.Unlock()
	if success {
		s.Cfg.RunTime.LastMairListCommState = fmt.Sprintf("Succeeded (%v)", time.Now().Format(dateFormat))
	} else {
		s.Cfg.RunTime.LastMairListCommState = fmt.Sprintf("Failed (%v)", time.Now().Format(dateFormat))
	}
}

// execRequest executes the request against mAirList and returns the data
func (s DefaultExportService) execRequest(req *http.Request) (respData []byte, respStatus int, err error) {
	resp, err := httpExClient.Do(req)
	if err != nil {
		s.SetMairListCommState(false)
		return nil, 0, err
	}
	if resp.StatusCode == 404 {
		s.SetMairListCommState(false)
		err := errors.New("url not found")
		return nil, resp.StatusCode, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		s.SetMairListCommState(false)
		return nil, resp.StatusCode, err
	}
	s.SetMairListCommState(true)
	return b, resp.StatusCode, nil
}

// buildAppendRequest is a helper function constructing the mAirList API request to append the playlist
func (s DefaultExportService) buildAppendRequest(fileName string) (req *http.Request, e error) {
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
	req, _ = http.NewRequest("POST", mairListUrl.String(), strings.NewReader(cmd.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(s.Cfg.Export.MairListUser, s.Cfg.Export.MairListPassword)
	return req, nil
}

// buildGetPlaylistRequest is a helper function constructing the mAirList API request the current playlist
func (s DefaultExportService) buildGetPlaylistRequest() (req *http.Request, e error) {
	if s.Cfg.Export.MairListUrl == "" {
		return nil, errors.New("url cannot be empty")
	}
	mairListUrl, err := url.Parse(s.Cfg.Export.MairListUrl)
	if err != nil {
		return nil, err
	}
	mairListUrl.Path = path.Join(mairListUrl.Path, "/playlist/0/content")
	req, _ = http.NewRequest("GET", mairListUrl.String(), nil)
	req.SetBasicAuth(s.Cfg.Export.MairListUser, s.Cfg.Export.MairListPassword)
	return req, nil
}

func (s DefaultExportService) GetPlaylist() error {
	// GET to http://<server>:9300/playlist/0/content
	// Basic Auth
	// Returns the current playlist as XML
	var (
		p bool
	)
	req, err := s.buildGetPlaylistRequest()
	if err != nil {
		return err
	}
	data, statusCode, err := s.execRequest(req)
	if err != nil {
		return err
	}
	if statusCode == 200 {
		if s.Cfg.Export.MairListVersion >= 6 {
			p, err = parseMairListPlaylistJson(data)
			if err != nil {
				return err
			}
		} else {
			p, err = parseMairListPlaylistXml(data)
			if err != nil {
				return err
			}
		}

		s.Cfg.RunTime.Mu.Lock()
		defer s.Cfg.RunTime.Mu.Unlock()
		s.Cfg.RunTime.MairListPlaying = p
		return nil
	}
	logger.Error("could not get mAirList playlist", err)
	return err
}

func parseMairListPlaylistXml(playlistData []byte) (playing bool, e error) {
	var (
		playList domain.MairListPlaylistXml
	)
	err := xml.Unmarshal(playlistData, &playList)
	if err != nil {
		logger.Error("Error converting mAirList playlist data into playlist", err)
		return false, err
	}
	for _, item := range playList.PlaylistItem {
		if item.State == "playing" && item.Class != "InfiniteSilence" {
			return true, nil
		}
	}
	return false, nil
}

func parseMairListPlaylistJson(playlistData []byte) (playing bool, e error) {
	var (
		playList domain.MairListPlaylistJson
	)
	err := json.Unmarshal(playlistData, &playList)
	if err != nil {
		logger.Error("Error converting mAirList playlist data into playlist", err)
		return false, err
	}
	for _, item := range playList.Items {
		if item.State == "playing" && item.Class != "InfiniteSilence" {
			return true, nil
		}
	}
	return false, nil
}

func (s DefaultExportService) QueryStatus() {
	logger.Info("Starting to query mAirList Playlist Status...")
	for {
		s.GetPlaylist()
		time.Sleep(time.Duration(s.Cfg.Export.StatusQueryCycleSec) * time.Second)
	}
}
