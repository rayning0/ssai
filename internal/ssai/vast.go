package ssai

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type VAST struct {
	XMLName xml.Name `xml:"VAST"`
	Ads     []VASTAd `xml:"Ad"`
}

type VASTAd struct {
	ID     string     `xml:"id,attr"`
	InLine VASTInLine `xml:"InLine"`
}

type VASTInLine struct {
	AdTitle     string         `xml:"AdTitle"`
	Impressions []string       `xml:"Impression"`
	Creatives   []VASTCreative `xml:"Creatives>Creative"`
}

type VASTCreative struct {
	Linear VASTLinear `xml:"Linear"`
}

type VASTLinear struct {
	Duration       string         `xml:"Duration"`
	TrackingEvents []VASTTracking `xml:"TrackingEvents>Tracking"`
	MediaFiles     []VASTMedia    `xml:"MediaFiles>MediaFile"`
}

type VASTTracking struct {
	Event string `xml:"event,attr"`
	URL   string `xml:",chardata"`
}

type VASTMedia struct {
	Type     string `xml:"type,attr"`
	Delivery string `xml:"delivery,attr"`
	URL      string `xml:",chardata"`
}

type SelectedAd struct {
	ID           string
	Duration     time.Duration
	MediaURL     string
	Impressions  []string
	TrackingURLs map[string][]string
}

func ParseVAST(data []byte) ([]SelectedAd, error) {
	var vast VAST

	if err := xml.Unmarshal(data, &vast); err != nil {
		return nil, fmt.Errorf("parse VAST XML: %w", err)
	}

	var ads []SelectedAd

	for _, ad := range vast.Ads {
		if len(ad.InLine.Creatives) == 0 {
			continue
		}

		linear := ad.InLine.Creatives[0].Linear

		duration, err := parseVASTDuration(linear.Duration)
		if err != nil {
			return nil, fmt.Errorf("ad %s: %w", ad.ID, err)
		}

		if len(linear.MediaFiles) == 0 {
			continue
		}

		selected := SelectedAd{
			ID:           ad.ID,
			Duration:     duration,
			MediaURL:     strings.TrimSpace(linear.MediaFiles[0].URL),
			TrackingURLs: make(map[string][]string),
		}

		for _, impression := range ad.InLine.Impressions {
			selected.Impressions = append(
				selected.Impressions,
				strings.TrimSpace(impression),
			)
		}

		for _, tracking := range linear.TrackingEvents {
			selected.TrackingURLs[tracking.Event] = append(
				selected.TrackingURLs[tracking.Event],
				strings.TrimSpace(tracking.URL),
			)
		}

		ads = append(ads, selected)
	}

	return ads, nil
}

func parseVASTDuration(value string) (time.Duration, error) {
	parts := strings.Split(strings.TrimSpace(value), ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid VAST duration: %q", value)
	}

	hours, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, err
	}

	minutes, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, err
	}

	seconds, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return 0, err
	}

	if hours < 0 || minutes < 0 || minutes >= 60 ||
		seconds < 0 || seconds >= 60 {
		return 0, fmt.Errorf("invalid VAST duration: %q", value)
	}

	return time.Duration(hours)*time.Hour +
		time.Duration(minutes)*time.Minute +
		time.Duration(seconds*float64(time.Second)), nil
}
