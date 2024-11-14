// package domain defines the core data structures
package domain

import "encoding/xml"

// Playlist is returned from mAirList via API
type MairListPlaylist struct {
	XMLName      xml.Name `xml:"Playlist"`
	Text         string   `xml:",chardata"`
	Version      string   `xml:"Version,attr"`
	PlaylistItem []struct {
		Text             string `xml:",chardata"`
		Class            string `xml:"Class,attr"`
		Version          string `xml:"Version,attr"`
		State            string `xml:"State,attr"`
		Time             string `xml:"Time,attr"`
		Player           string `xml:"Player,attr"`
		PlaybackPosition string `xml:"PlaybackPosition,attr"`
		PlaybackEnd      string `xml:"PlaybackEnd,attr"`
		DisplayEnd       string `xml:"DisplayEnd,attr"`
		Filename         string `xml:"Filename"`
		Title            string `xml:"Title"`
		Artist           string `xml:"Artist"`
		Type             string `xml:"Type"`
		Duration         string `xml:"Duration"`
		Database         string `xml:"Database"`
		DatabaseID       string `xml:"DatabaseID"`
		Attributes       struct {
			Text string `xml:",chardata"`
			Item []struct {
				Text  string `xml:",chardata"`
				Name  string `xml:"Name"`
				Value string `xml:"Value"`
			} `xml:"Item"`
		} `xml:"Attributes"`
		Markers struct {
			Text   string `xml:",chardata"`
			Marker []struct {
				Text     string `xml:",chardata"`
				Type     string `xml:"Type,attr"`
				Position string `xml:"Position,attr"`
			} `xml:"Marker"`
		} `xml:"Markers"`
		Amplification string `xml:"Amplification"`
	} `xml:"PlaylistItem"`
}
