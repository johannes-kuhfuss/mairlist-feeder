// package dto defines the data structures used to exchange information
package dto

// Event defines the data used to display the events
type Event struct {
	CurrentEvent    string `json:"current_event"`
	EventId         string `json:"event_id"`
	Title           string `json:"title"`
	StartDate       string `json:"start_date"`
	StartTime       string `json:"start_time"`
	EndTime         string `json:"end_time"`
	PlannedDuration string `json:"planned_duration"`
	ActualDuration  string `json:"actual_duration"`
	EventType       string `json:"event_type"`
	FileStatus      string `json:"file_status"`
	FileSource      string `json:"file_source"`
	FileAvail       string `json:"file_avail"`
}
