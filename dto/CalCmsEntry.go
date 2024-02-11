package dto

import "time"

type CalCmsEntry struct {
	Title     string
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
	EventId   int
	Live      int
}
