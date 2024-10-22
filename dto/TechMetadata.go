// package dto defines the data structures used to exchange information
package dto

// TechnicalMetadata defines the data retrived from ffprobe
type TechnicalMetadata struct {
	DurationSec float64
	BitRate     int64
	FormatName  string
}
