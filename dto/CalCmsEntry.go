// package dto defines the data structures used to exchange information
package dto

import "time"

// CalCmsEntry defines a subset of data from calCms used to describe a calCms event
type CalCmsEntry struct {
	Title     string
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
	EventId   int
	Live      int
}
