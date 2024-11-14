// package dto defines the data structures used to exchange information
package dto

// MairListRequest describes the request sent to mAirList
// There are 3 requests: Get On-Air Status, Append Playlist, Get Current Playlist
// ReqType = onair, appendpl, getpl
// For Append Playlist, a file name needs to be passed
type MairListRequest struct {
	ReqType  string
	FileName string
}
