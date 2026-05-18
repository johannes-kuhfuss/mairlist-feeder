// package dto defines the data structures used to exchange information
package dto

import "time"

// TechnicalMetadata defines the data retrived from ffprobe
type TechnicalMetadata struct {
	Duration   time.Duration
	BitRate    int64
	FormatName string
}
