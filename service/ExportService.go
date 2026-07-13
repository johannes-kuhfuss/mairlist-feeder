// package service implements the services and their business logic that provide the main part of the program
package service

import (
	"bufio"
	"context"
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

	"github.com/johannes-kuhfuss/mairlist-feeder/appstate"
	"github.com/johannes-kuhfuss/mairlist-feeder/config"
	"github.com/johannes-kuhfuss/mairlist-feeder/domain"
	"github.com/johannes-kuhfuss/mairlist-feeder/dto"
	"github.com/johannes-kuhfuss/mairlist-feeder/helper"
	"github.com/johannes-kuhfuss/mairlist-feeder/repositories"
	"github.com/johannes-kuhfuss/services_utils/logger"
)

type Exporter interface {
	Export() error
	ExportContext(context.Context) error
}

const (
	dateFormat = "2006-01-02 15:04:05 -0700 MST"
)

// The export service handles the export of information to mAirList
type DefaultExportService struct {
	Cfg        *config.AppConfig
	State      *appstate.AppState
	Repo       *repositories.DefaultFileRepository
	httpClient *http.Client
	Now        func() time.Time
	mu         *sync.Mutex
}

type exportPlan map[string]domain.FileInfo

// InitHttpExClient sets the default values for the http client used to interact with mAirlist
func InitHttpExClient() *http.Client {
	httpExTr := http.Transport{
		DisableKeepAlives:  false,
		DisableCompression: false,
		MaxIdleConns:       0,
		IdleConnTimeout:    0,
	}
	return &http.Client{
		Transport: &httpExTr,
		Timeout:   5 * time.Second,
	}
}

// NewExportService creates a new export service and injects its dependencies
func NewExportService(cfg *config.AppConfig, repo *repositories.DefaultFileRepository) DefaultExportService {
	return NewExportServiceWithState(cfg, appstate.New(), repo)
}

func NewExportServiceWithState(cfg *config.AppConfig, state *appstate.AppState, repo *repositories.DefaultFileRepository) DefaultExportService {
	return DefaultExportService{
		Cfg:        cfg,
		State:      state,
		Repo:       repo,
		httpClient: InitHttpExClient(),
		Now:        time.Now,
		mu:         &sync.Mutex{},
	}
}

// Export orchestrates the export of data to mAirList
func (s DefaultExportService) Export() (err error) {
	return s.ExportContext(context.Background())
}

func (s DefaultExportService) ExportContext(ctx context.Context) (err error) {
	start := s.Now()
	defer func() {
		recordRunMetrics(s.State, "export", start, err)
	}()
	s.State.Runtime.Update(func(runtime *appstate.RuntimeState) { runtime.LastExportRunDate = s.Now() })
	exportDate, nextHour := getNextExportSlot(s.Now())
	return s.ExportForDateAndHourContext(ctx, exportDate, nextHour)
}

// ExportAllHours exports a playlist for all hours of the day
func (s DefaultExportService) ExportAllHours() (err error) {
	return s.ExportAllHoursContext(context.Background())
}

func (s DefaultExportService) ExportAllHoursContext(ctx context.Context) (err error) {
	start := s.Now()
	defer func() {
		recordRunMetrics(s.State, "export", start, err)
	}()
	return s.ExportAllHoursForDateContext(ctx, helper.DateForFolder(s.Cfg.Misc.TestCrawl, s.Cfg.Misc.TestDate, 0))
}

// ExportAllHoursForDate exports a playlist for all hours of a specific day.
func (s DefaultExportService) ExportAllHoursForDate(folderDate time.Time) error {
	return s.ExportAllHoursForDateContext(context.Background(), folderDate)
}

func (s DefaultExportService) ExportAllHoursForDateContext(ctx context.Context, folderDate time.Time) error {
	var runErr error
	for hour := range 24 {
		if err := ctx.Err(); err != nil {
			return errors.Join(runErr, err)
		}
		runErr = errors.Join(runErr, s.ExportForDateAndHourContext(ctx, folderDate, fmt.Sprintf("%02d", hour)))
	}
	return runErr
}

// ExportForHour exports a playlist for a given hour
// Loads playlist into mAirList via API, if enabled
func (s DefaultExportService) ExportForHour(hour string) (err error) {
	return s.ExportForHourContext(context.Background(), hour)
}

func (s DefaultExportService) ExportForHourContext(ctx context.Context, hour string) (err error) {
	start := s.Now()
	defer func() {
		recordRunMetrics(s.State, "export", start, err)
	}()
	return s.ExportForDateAndHourContext(ctx, helper.DateForFolder(s.Cfg.Misc.TestCrawl, s.Cfg.Misc.TestDate, 0), hour)
}

// ExportForDateAndHour exports a playlist for a given folder date and hour.
// Loads playlist into mAirList via API, if enabled.
func (s DefaultExportService) ExportForDateAndHour(folderDate time.Time, hour string) error {
	return s.ExportForDateAndHourContext(context.Background(), folderDate, hour)
}

func (s DefaultExportService) ExportForDateAndHourContext(ctx context.Context, folderDate time.Time, hour string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.State.Runtime.Update(func(runtime *appstate.RuntimeState) { runtime.ExportRunning = true })
	defer func() {
		s.State.Runtime.Update(func(runtime *appstate.RuntimeState) { runtime.ExportRunning = false })
	}()
	if files := s.Repo.GetByDateAndHour(folderDate, hour, s.Cfg.Export.ExportLiveItems); files != nil {
		logger.Infof("Starting export for %v %v:00 ...", domain.FormatFolderDate(folderDate), hour)
		start := s.Now().UTC()
		sort.Sort(files)
		plan := s.checkTimeAndLength(files)
		exportPath, err := s.exportToPlayoutForDate(folderDate, hour, plan)
		if s.Cfg.Export.AppendPlaylist && exportPath != "" && err == nil {
			req := dto.MairListRequest{
				ReqType:  dto.MairListRequestAppendPlaylist,
				FileName: exportPath,
			}
			err := s.ExecuteMairListRequestContext(ctx, req)
			if err != nil {
				logger.Error("Error appending playlist", err)
				return err
			}
		}
		if err != nil {
			return err
		}
		end := s.Now().UTC()
		dur := end.Sub(start)
		logger.Infof("Finished exporting for %v %v:00 ... (%v)", domain.FormatFolderDate(folderDate), hour, dur.String())
	} else {
		logger.Infof("No files to export for %v %v:00 ...", domain.FormatFolderDate(folderDate), hour)
	}
	return nil
}

// checkTimeAndLength determines suitability of files for playout, based on their length
// Also resolves conflicts if there are multiple matching files for the same time
func (s DefaultExportService) checkTimeAndLength(files domain.FileList) exportPlan {
	plan := make(exportPlan)
	for _, file := range files {
		lengthOk, slotLen, info := checkTime(file, s.Cfg.Export.ShortDeltaAllowance, s.Cfg.Export.LongDeltaAllowance)
		logger.Infof("File: %v, ModDate: %v, IsOK: %v, Info: %v", file.Path, file.ModTime, lengthOk, info)
		if lengthOk {
			file.SlotLength = slotLen
			preFile, exists := plan[createIndexFromTime(file.StartTime)]
			if exists {
				if preFile.ModTime.After(file.ModTime) {
					logger.Infof("Existing file %v is newer than file %v. Not updating.", preFile.Path, file.Path)
				} else {
					logger.Infof("Existing file %v is older than file %v. Updating.", preFile.Path, file.Path)
					plan[createIndexFromTime(file.StartTime)] = file
				}
			} else {
				plan[createIndexFromTime(file.StartTime)] = file
			}
		}
	}
	return plan
}

// getNextHour is a helper function that returns the next hour
func getNextHour() string {
	_, nextHour := getNextExportSlot(time.Now())
	return nextHour
}

func getNextExportSlot(now time.Time) (time.Time, string) {
	next := now.Add(time.Hour)
	return domain.NormalizeDate(next), fmt.Sprintf("%02d", next.Hour())
}

// createIndexFromTime is a helper function that returns the time in HH:MM format
func createIndexFromTime(t1 time.Time) string {
	return t1.Format("15:04")
}

// checkTime is a helper function that compares available time information such as start time, end time, etc.
// It classifies the entry into a slot length and calculates differences between the actual file length and the presumed slot length
// Finally it makes a determination whether the file's length is OK to play the file
func checkTime(fi domain.FileInfo, shortDelta float64, longDelta float64) (lengthOk bool, slot time.Duration, info string) {
	var (
		lengthSlot   time.Duration
		slotDelta    float64
		plannedDur   float64
		durDelta     float64
		plannedAvail bool
		detail       string
		lenStr       string
	)
	roundedDurationMin := math.Round(fi.Duration.Minutes())
	is30Min := (roundedDurationMin >= 30.0-shortDelta) && (roundedDurationMin <= 30.0+longDelta)
	is45Min := (roundedDurationMin >= 45.0-shortDelta) && (roundedDurationMin <= 45.0+longDelta)
	is60Min := (roundedDurationMin >= 60.0-shortDelta) && (roundedDurationMin <= 60.0+longDelta)
	is90Min := (roundedDurationMin >= 90.0-shortDelta) && (roundedDurationMin <= 90.0+longDelta)
	is120Min := (roundedDurationMin >= 120.0-shortDelta) && (roundedDurationMin <= 120.0+longDelta)
	isLonger := (roundedDurationMin > 120.0+longDelta)
	switch {
	case is30Min:
		lengthSlot = 30 * time.Minute
		slotDelta = roundedDurationMin - 30.0
	case is45Min:
		lengthSlot = 45 * time.Minute
		slotDelta = roundedDurationMin - 45.0
	case is60Min:
		lengthSlot = 60 * time.Minute
		slotDelta = roundedDurationMin - 60.0
	case is90Min:
		lengthSlot = 90 * time.Minute
		slotDelta = roundedDurationMin - 90.0
	case is120Min:
		lengthSlot = 120 * time.Minute
		slotDelta = roundedDurationMin - 120.0
	case isLonger:
		lengthSlot = time.Duration(roundedDurationMin) * time.Minute
		slotDelta = 0.0
		logger.Warnf("Detected very long file: %v with length %vmin. Please verify.", fi.Path, roundedDurationMin)
	default:
		lengthSlot = 0
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
	if lengthSlot > 0 {
		lenStr = strconv.Itoa(int(math.Round(lengthSlot.Minutes()))) + "min"

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
	return s.ExportToPlayoutForDate(helper.DateForFolder(s.Cfg.Misc.TestCrawl, s.Cfg.Misc.TestDate, 0), hour)
}

// ExportToPlayoutForDate writes a ".tpi" playlist to disk for a given date and hour.
func (s DefaultExportService) ExportToPlayoutForDate(folderDate time.Time, hour string) (exportedFile string, err error) {
	if files := s.Repo.GetByDateAndHour(folderDate, hour, s.Cfg.Export.ExportLiveItems); files != nil {
		sort.Sort(files)
		return s.exportToPlayoutForDate(folderDate, hour, s.checkTimeAndLength(files))
	}
	return "", nil
}

func (s DefaultExportService) exportToPlayoutForDate(folderDate time.Time, hour string, plan exportPlan) (exportedFile string, err error) {
	// write export list ot mAirlist-compatible file
	// Documentation: https://wiki.mairlist.com/reference:text_playlist_import_format_specification
	// Tab separated
	// Column layout:
	/// 1 - start time = HH:MM
	/// 2 - timing = H (hard fixed time), N (normal)
	/// 3 - line type = F (file), I (database item)
	/// 4 - Line data = full path file name, database Id
	/// 5 - Optional values = omitted here
	if size := len(plan); size > 0 {
		logger.Infof("Exporting %v elements to mAirList for slot %v %v:00", size, domain.FormatFolderDate(folderDate), hour)
		exportPath, err := s.setExportPathForDate(folderDate, hour)
		if err != nil {
			logger.Error("Error when setting export path", err)
			return "", err
		}
		if err := s.WritePlaylist(exportPath, plan); err != nil {
			return "", err
		}
		s.State.Runtime.Update(func(runtime *appstate.RuntimeState) {
			runtime.LastExportFileName = exportPath
			runtime.LastExportedFileDate = s.Now()
		})
		return exportPath, nil
	}
	logger.Infof("No elements to export for slot %v %v:00.", domain.FormatFolderDate(folderDate), hour)
	return "", nil
}

func (s DefaultExportService) WritePlaylist(exportPath string, plan exportPlan) error {
	var (
		totalLength time.Duration
		startTime   time.Time
		line        string
	)
	exportDir := filepath.Dir(exportPath)
	exportFile, err := os.CreateTemp(exportDir, filepath.Base(exportPath)+".*.tmp")
	if err != nil {
		logger.Error("Error when creating playlist file for mAirlist", err)
		return err
	}
	tmpPath := exportFile.Name()
	defer os.Remove(tmpPath)
	closeWithError := func(currentErr error) error {
		if closeErr := exportFile.Close(); currentErr == nil {
			return closeErr
		}
		return currentErr
	}
	dataWriter := bufio.NewWriter(exportFile)
	if err := s.writeStartComment(dataWriter); err != nil {
		return closeWithError(err)
	}
	keys := make([]string, 0, len(plan))
	for timeKey := range plan {
		keys = append(keys, timeKey)
	}
	sort.Strings(keys)
	for _, timeKey := range keys {
		file := plan[timeKey]
		startTime = setStartTime(startTime, timeKey)
		if file.FromCalCMS && !file.EndTime.IsZero() {
			plannedDur := file.EndTime.Sub(file.StartTime)
			totalLength = totalLength + plannedDur
		} else {
			totalLength = totalLength + file.SlotLength
		}
		listTime := timeKey + ":00"
		switch file.FileType {
		case domain.FileTypeStream:
			line = fmt.Sprintf("%v\tH\tI\t%v\n", listTime, file.StreamId)
		default:
			line = fmt.Sprintf("%v\tH\tF\t%v\n", listTime, file.Path)
		}
		if err := s.writeLine(dataWriter, line); err != nil {
			return closeWithError(err)
		}
	}
	if s.Cfg.Export.TerminateAfterDuration {
		if err := s.WriteStopper(dataWriter, startTime, totalLength); err != nil {
			return closeWithError(err)
		}
	}
	if err := s.writeEndComment(dataWriter); err != nil {
		return closeWithError(err)
	}
	if err := dataWriter.Flush(); err != nil {
		logger.Error("Error flushing playlist file for mAirlist", err)
		return closeWithError(err)
	}
	if err := exportFile.Sync(); err != nil {
		return closeWithError(err)
	}
	if err := exportFile.Chmod(0644); err != nil {
		return closeWithError(err)
	}
	if err := exportFile.Close(); err != nil {
		return err
	}
	if err := replaceFile(tmpPath, exportPath); err != nil {
		return err
	}
	return nil
}

// replaceFile installs a fully-written temporary file. On platforms where rename
// cannot replace an existing file, the old file is restored if installation fails.
func replaceFile(tmpPath, destination string) error {
	if err := os.Rename(tmpPath, destination); err == nil {
		return nil
	}
	backupFile, err := os.CreateTemp(filepath.Dir(destination), filepath.Base(destination)+".*.bak")
	if err != nil {
		return err
	}
	backupPath := backupFile.Name()
	if err := backupFile.Close(); err != nil {
		return err
	}
	if err := os.Remove(backupPath); err != nil {
		return err
	}
	destinationMoved := false
	if err := os.Rename(destination, backupPath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	} else {
		destinationMoved = true
	}
	if err := os.Rename(tmpPath, destination); err != nil {
		if destinationMoved {
			_ = os.Rename(backupPath, destination)
		}
		return err
	}
	if destinationMoved {
		_ = os.Remove(backupPath)
	}
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
	return s.setExportPathForDate(helper.DateForFolder(s.Cfg.Misc.TestCrawl, s.Cfg.Misc.TestDate, 0), hour)
}

func (s DefaultExportService) setExportPathForDate(folderDate time.Time, hour string) (exportPath string, e error) {
	var exportFileName string
	if s.Cfg.Misc.TestCrawl {
		exportFileName = "Test_" + hour + ".tpi"
	} else {
		exportFileName = domain.FormatFolderDate(folderDate) + "-" + hour + ".tpi"
	}
	expPath := filepath.Join(s.Cfg.Export.ExportFolder, exportFileName)
	absExpPath, err := filepath.Abs(expPath)
	if err != nil {
		return "", err
	}
	if !helper.IsPathWithin(absExpPath, s.Cfg.Export.ExportFolder) {
		return "", errors.New("invalid export path")
	}
	return absExpPath, nil
}

// writeStartComment is a helper function creating the ".tpi" file's start comment
func (s DefaultExportService) writeStartComment(w *bufio.Writer) error {
	line := fmt.Sprintf("\t\tR\tPlaylist auto-generated by mAirList Feeder at %v\n", s.Now().Format("2006-01-02 15:04:05"))
	return s.writeLine(w, line)
}

// WriteStopper is a helper function adding an end element to the ".tpi" file
func (s DefaultExportService) WriteStopper(w *bufio.Writer, startTime time.Time, tlen time.Duration) error {
	eh := startTime.Add(tlen)
	line := fmt.Sprintf("%v\tH\tD\tEnd of block\n", eh.Format("15:04:05"))
	return s.writeLine(w, line)
}

// writeEndComment is a helper function creating the ".tpi" file's end comment
func (s DefaultExportService) writeEndComment(w *bufio.Writer) error {
	line := "\t\tR\tEnd of auto-generated playlist\n"
	return s.writeLine(w, line)
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
	return s.ExecuteMairListRequestContext(context.Background(), req)
}

func (s DefaultExportService) ExecuteMairListRequestContext(ctx context.Context, req dto.MairListRequest) error {
	// ReqType = appendpl, getpl
	switch req.ReqType {
	case dto.MairListRequestAppendPlaylist:
		err := s.AppendPlaylistContext(ctx, req.FileName)
		if err != nil {
			return err
		}
	case dto.MairListRequestGetPlaylist:
		err := s.GetPlaylistContext(ctx)
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
	return s.AppendPlaylistContext(context.Background(), fileName)
}

func (s DefaultExportService) AppendPlaylistContext(ctx context.Context, fileName string) error {
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
	req = req.WithContext(ctx)
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
	s.State.Runtime.Update(func(runtime *appstate.RuntimeState) {
		if success {
			runtime.LastMairListCommState = fmt.Sprintf("Succeeded (%v)", s.Now().Format(dateFormat))
		} else {
			runtime.LastMairListCommState = fmt.Sprintf("Failed (%v)", s.Now().Format(dateFormat))
		}
	})
	if success {
		s.State.Metrics.SetConnected("mAirList", 1)
	} else {
		s.State.Metrics.SetConnected("mAirList", 0)
	}
}

// execRequest executes the request against mAirList and returns the data
func (s DefaultExportService) execRequest(req *http.Request) (respData []byte, respStatus int, err error) {
	resp, err := s.httpClient.Do(req)
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
	req, err = http.NewRequest(http.MethodPost, mairListUrl.String(), strings.NewReader(cmd.Encode()))
	if err != nil {
		return nil, err
	}
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
	req, err = http.NewRequest(http.MethodGet, mairListUrl.String(), nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(s.Cfg.Export.MairListUser, s.Cfg.Export.MairListPassword)
	return req, nil
}

func (s DefaultExportService) GetPlaylist() error {
	return s.GetPlaylistContext(context.Background())
}

func (s DefaultExportService) GetPlaylistContext(ctx context.Context) error {
	// GET to http://<server>:9300/playlist/0/content
	// Basic Auth
	// Returns the current playlist as XML
	var (
		playing bool
	)
	req, err := s.buildGetPlaylistRequest()
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)
	data, statusCode, err := s.execRequest(req)
	if err != nil {
		return err
	}
	if statusCode == 200 {
		if s.Cfg.Export.MairListVersion >= 6 {
			playing, err = parseMairListPlaylistJson(data)
			if err != nil {
				return err
			}
		} else {
			playing, err = parseMairListPlaylistXml(data)
			if err != nil {
				return err
			}
		}
		if playing {
			s.State.Metrics.SetMairListPlaying(s.Cfg.Export.MairListUrl, 1)
		} else {
			s.State.Metrics.SetMairListPlaying(s.Cfg.Export.MairListUrl, 0)
		}
		s.State.Runtime.Update(func(runtime *appstate.RuntimeState) { runtime.MairListPlaying = playing })
		return nil
	}
	err = fmt.Errorf("mAirList playlist request failed: HTTP %d", statusCode)
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

func (s DefaultExportService) QueryStatus(ctx context.Context) {
	logger.Info("Starting to query mAirList Playlist Status...")
	ticker := time.NewTicker(time.Duration(s.Cfg.Export.StatusQueryCycleSec) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			logger.Info("Stopped querying mAirList Playlist Status")
			return
		default:
			if err := s.GetPlaylistContext(ctx); err != nil && ctx.Err() == nil {
				logger.Error("could not query mAirList playlist status", err)
			}
		}
		select {
		case <-ctx.Done():
			logger.Info("Stopped querying mAirList Playlist Status")
			return
		case <-ticker.C:
		}
	}
}
