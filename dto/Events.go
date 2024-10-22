// package dto defines the data structures used to exchange information
package dto

// Event defines the data used to display the day's events 
type Event struct {
	EventId         string
	Title           string
	StartDate       string
	StartTime       string
	EndTime         string
	PlannedDuration string
	ActualDuration  string
	EventType       string
	FileStatus      string
}
