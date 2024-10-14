package dto

type Event struct {
	EventId     string
	Title       string
	StartDate   string
	StartTime   string
	EndTime     string
	Duration    string
	LiveEvent   bool
	FilePresent bool
}
