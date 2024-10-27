// package domain defines the core data structures
package domain

// CalCmsPgmData is the data structure returned from calCms
type CalCmsPgmData struct {
	Archive     string `json:"archive"`
	BaseDomain  string `json:"base_domain"`
	BaseURL     string `json:"base_url"`
	Controllers struct {
		Atom     string `json:"atom"`
		Calendar string `json:"calendar"`
		Comments string `json:"comments"`
		Domain   string `json:"domain"`
		Event    string `json:"event"`
		Events   string `json:"events"`
		Ical     string `json:"ical"`
		Rss      string `json:"rss"`
	} `json:"controllers"`
	Date             string `json:"date"`
	DateRangeInclude int    `json:"date_range_include"`
	DefaultProject   struct {
		Email     string `json:"email"`
		EndDate   string `json:"end_date"`
		Image     string `json:"image"`
		Name      string `json:"name"`
		ProjectID int    `json:"project_id"`
		StartDate string `json:"start_date"`
		Subtitle  string `json:"subtitle"`
		Title     string `json:"title"`
	} `json:"default_project"`
	DisableEventSync   string        `json:"disable_event_sync"`
	EditorBaseURL      string        `json:"editor_base_url"`
	EventCount         string        `json:"event_count"`
	EventDtstart       string        `json:"event_dtstart"`
	EventID            int           `json:"event_id"`
	Events             []CalCmsEvent `json:"events"`
	EventsDescription  string        `json:"events_description"`
	EventsTitle        string        `json:"events_title"`
	ExcludeEventImages int           `json:"exclude_event_images"`
	ExcludeLocations   int           `json:"exclude_locations"`
	ExcludeProjects    int           `json:"exclude_projects"`
	Extern             int           `json:"extern"`
	FirstDate          string        `json:"first_date"`
	FirstOfList        int           `json:"first_of_list"`
	FromDate           string        `json:"from_date"`
	FromTime           string        `json:"from_time"`
	Get                string        `json:"get"`
	IconsURL           string        `json:"icons_url"`
	ImagesURL          string        `json:"images_url"`
	JSONCallback       string        `json:"json_callback"`
	LastDate           string        `json:"last_date"`
	LastDays           int           `json:"last_days"`
	Limit              string        `json:"limit"`
	ListenURL          string        `json:"listen_url"`
	LocalBaseURL       string        `json:"local_base_url"`
	Location           string        `json:"location"`
	LocationsToExclude string        `json:"locations_to_exclude"`
	ModifiedAt         string        `json:"modified_at"`
	Month              string        `json:"month"`
	NoResult           any           `json:"no_result"`
	OnlyRecordings     string        `json:"only_recordings"`
	Order              string        `json:"order"`
	Project            string        `json:"project"`
	ProjectColoradio   int           `json:"project_coloradio"`
	ProjectEmail       string        `json:"project_email"`
	ProjectEndDate     string        `json:"project_end_date"`
	ProjectID          string        `json:"project_id"`
	ProjectImage       string        `json:"project_image"`
	ProjectName        string        `json:"project_name"`
	ProjectProjectID   int           `json:"project_project_id"`
	ProjectStartDate   string        `json:"project_start_date"`
	ProjectSubtitle    string        `json:"project_subtitle"`
	ProjectTitle       string        `json:"project_title"`
	ProjectsToExclude  string        `json:"projects_to_exclude"`
	Recordings         int           `json:"recordings"`
	Ro                 int           `json:"ro"`
	Search             string        `json:"search"`
	SeriesName         string        `json:"series_name"`
	SetNoListenKeys    int           `json:"set_no_listen_keys"`
	SourceBaseURL      string        `json:"source_base_url"`
	StaticFilesURL     string        `json:"static_files_url"`
	StudioID           string        `json:"studio_id"`
	StudioName         string        `json:"studio_name"`
	Tag                string        `json:"tag"`
	Template           string        `json:"template"`
	ThumbsURL          string        `json:"thumbs_url"`
	TillDate           string        `json:"till_date"`
	TillTime           string        `json:"till_time"`
	Time               string        `json:"time"`
	TimeZone           string        `json:"time_zone"`
	Title              string        `json:"title"`
	User               any           `json:"user"`
	UtcOffset          string        `json:"utc_offset"`
	Weekday            string        `json:"weekday"`
	WidgetRenderURL    string        `json:"widget_render_url"`
}

// CalCmsEvent defines the subset of calCms data relevant for one program event
type CalCmsEvent struct {
	First                int    `json:"__first__,omitempty"`
	ArchiveURL           string `json:"archive_url"`
	Archived             int    `json:"archived"`
	BaseDomain           string `json:"base_domain"`
	BaseURL              any    `json:"base_url"`
	CommentCount         int    `json:"comment_count"`
	Content              string `json:"content"`
	ContentFormat        any    `json:"content_format"`
	Counter1             int    `json:"counter_1,omitempty"`
	CreatedAt            string `json:"created_at"`
	Day                  string `json:"day"`
	DayOfYear            int    `json:"day_of_year"`
	DisableEventSync     int    `json:"disable_event_sync"`
	Draft                int    `json:"draft"`
	Dtend                string `json:"dtend"`
	Dtstart              string `json:"dtstart"`
	Duration             string `json:"duration"`
	Ekey                 string `json:"ekey"`
	End                  string `json:"end"`
	EndDate              string `json:"end_date"`
	EndDateName          string `json:"end_date_name"`
	EndDatetime          string `json:"end_datetime"`
	EndDatetimeUtc       string `json:"end_datetime_utc"`
	EndTime              string `json:"end_time"`
	EndTimeName          string `json:"end_time_name"`
	EndUtcEpoch          int    `json:"end_utc_epoch"`
	Episode              any    `json:"episode"`
	EventID              int    `json:"event_id"`
	EventURI             string `json:"event_uri"`
	Excerpt              string `json:"excerpt"`
	FullTitle            string `json:"full_title"`
	FullTitleNoSeries    string `json:"full_title_no_series"`
	HTMLContent          string `json:"html_content"`
	HTMLTopic            string `json:"html_topic"`
	IconURL              string `json:"icon_url"`
	Image                string `json:"image"`
	ImageLabel           any    `json:"image_label"`
	ImageURL             string `json:"image_url"`
	IsFirstOfDay         int    `json:"is_first_of_day,omitempty"`
	ListenKey            any    `json:"listen_key"`
	Live                 int    `json:"live"`
	LocalBaseURL         string `json:"local_base_url"`
	Location             string `json:"location"`
	LocationCSS          string `json:"location_css"`
	LocationLabelStudio  int    `json:"location_label_studio"`
	MediaURL             any    `json:"media_url"`
	ModifiedAt           string `json:"modified_at"`
	ModifiedBy           string `json:"modified_by"`
	NoComment            int    `json:"no_comment"`
	NoImageInText        int    `json:"no_image_in_text"`
	Playout              int    `json:"playout"`
	PodcastURL           string `json:"podcast_url"`
	Program              any    `json:"program"`
	Project              string `json:"project"`
	ProjectEmail         string `json:"project_email"`
	ProjectEndDate       string `json:"project_end_date"`
	ProjectImage         string `json:"project_image"`
	ProjectName          string `json:"project_name"`
	ProjectProjectID     int    `json:"project_project_id"`
	ProjectStartDate     string `json:"project_start_date"`
	ProjectSubtitle      string `json:"project_subtitle"`
	ProjectTitle         string `json:"project_title"`
	Published            int    `json:"published"`
	RdsTitle             string `json:"rds_title"`
	Recurrence           int    `json:"recurrence"`
	RecurrenceCount      string `json:"recurrence_count"`
	RecurrenceCountAlpha string `json:"recurrence_count_alpha"`
	Reference            any    `json:"reference"`
	Rerun                any    `json:"rerun"`
	SeriesIconURL        string `json:"series_icon_url,omitempty"`
	SeriesImage          string `json:"series_image"`
	SeriesImageLabel     any    `json:"series_image_label"`
	SeriesImageURL       string `json:"series_image_url,omitempty"`
	SeriesName           string `json:"series_name"`
	SeriesThumbURL       string `json:"series_thumb_url,omitempty"`
	Skey                 string `json:"skey"`
	SourceBaseURL        string `json:"source_base_url"`
	Start                string `json:"start"`
	StartDate            string `json:"start_date"`
	StartDateName        string `json:"start_date_name"`
	StartDatetime        string `json:"start_datetime"`
	StartDatetimeUtc     string `json:"start_datetime_utc"`
	StartDay             string `json:"start_day"`
	StartHour            string `json:"start_hour"`
	StartMinute          string `json:"start_minute"`
	StartMonth           string `json:"start_month"`
	StartTime            string `json:"start_time"`
	StartTimeName        string `json:"start_time_name"`
	StartUtcEpoch        int    `json:"start_utc_epoch"`
	StartYear            string `json:"start_year"`
	StaticFilesURL       string `json:"static_files_url"`
	Status               any    `json:"status"`
	Stkey                string `json:"stkey"`
	ThumbURL             string `json:"thumb_url"`
	TimeZone             string `json:"time_zone"`
	Title                string `json:"title"`
	Tkey                 string `json:"tkey"`
	Topic                string `json:"topic"`
	UploadStatus         any    `json:"upload_status"`
	UserExcerpt          any    `json:"user_excerpt"`
	UserTitle            string `json:"user_title"`
	UtcOffset            string `json:"utc_offset"`
	WeekOfYear           int    `json:"week_of_year"`
	Weekday              int    `json:"weekday"`
	WeekdayName          string `json:"weekday_name"`
	WeekdayShortName     string `json:"weekday_short_name"`
	WidgetRenderURL      string `json:"widget_render_url"`
	Counter2             int    `json:"counter_2,omitempty"`
	Counter3             int    `json:"counter_3,omitempty"`
	IsRunning            int    `json:"is_running,omitempty"`
	Counter4             int    `json:"counter_4,omitempty"`
	Counter5             int    `json:"counter_5,omitempty"`
	Counter6             int    `json:"counter_6,omitempty"`
	Counter7             int    `json:"counter_7,omitempty"`
	Counter8             int    `json:"counter_8,omitempty"`
	Counter9             int    `json:"counter_9,omitempty"`
	Counter10            int    `json:"counter_10,omitempty"`
	IsLastOfDay          int    `json:"is_last_of_day,omitempty"`
	Counter11            int    `json:"counter_11,omitempty"`
	Last                 int    `json:"__last__,omitempty"`
	Counter12            int    `json:"counter_12,omitempty"`
}
