package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/rayning0/ssai/internal/hls"
	"github.com/rayning0/ssai/internal/ssai"
)

func main() {
	file, err := os.Open("testdata/sample-scte-playlist.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	snapshots, err := hls.ReadSnapshots(file)
	if err != nil {
		log.Fatal(err)
	}

	channel := ssai.NewChannelState("wcvb")

	fmt.Printf("Loaded %d HLS snapshots\n", len(snapshots))

	for i, snapshot := range snapshots {
		playlist, err := hls.ParsePlaylist(snapshot.Manifest)
		if err != nil {
			log.Fatalf("parsing snapshot %d: %v", i, err)
		}

		before := len(channel.Segments)

		if err := channel.Ingest(playlist); err != nil {
			log.Fatalf("ingesting snapshot %d: %v", i, err)
		}

		fmt.Printf(
			"Snapshot %2d: received=%2d new=%2d total=%2d\n",
			i,
			len(playlist.Segments),
			len(channel.Segments)-before,
			len(channel.Segments),
		)
	}

	fmt.Println("\nAd breaks:")

	for _, b := range channel.AdBreaks {
		fmt.Printf(
			"ID=%s duration=%.3f observedThrough=%.3f "+
				"remaining=%.3f ended=%t firstObservedSeq=%d\n",
			b.ID,
			b.Duration,
			b.ObservedThrough,
			b.Remaining(),
			b.Ended,
			b.FirstObservedSequence,
		)
	}

	fmt.Printf(
		"\nUnique segments: %d\nLast sequence: %d\n",
		len(channel.Segments),
		channel.LastSequence,
	)

	vastXML, err := os.ReadFile("testdata/mock-vast.xml")
	if err != nil {
		log.Fatal(err)
	}

	ads, err := ssai.ParseVAST(vastXML)
	if err != nil {
		log.Fatal(err)
	}

	adServer := &ssai.MockAdServer{Ads: ads}

	var breakID string

	for id := range channel.AdBreaks {
		breakID = id
		break
	}

	if breakID == "" {
		log.Fatal("no ad break found")
	}

	for _, remaining := range []time.Duration{
		90 * time.Second,
		8 * time.Second,
		3 * time.Second,
	} {
		decision, err := adServer.Decide(ssai.AdRequest{
			SessionID:         "viewer123",
			BreakID:           breakID,
			RemainingDuration: remaining,
		})
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf(
			"Remaining=%s selectedAds=%d\n",
			remaining,
			len(decision.Ads),
		)

		for _, ad := range decision.Ads {
			fmt.Printf(
				"  Ad=%s duration=%s media=%s\n",
				ad.ID,
				ad.Duration,
				ad.MediaURL,
			)

			fmt.Printf(
				"  Impressions=%v\n  Tracking=%v\n",
				ad.Impressions,
				ad.TrackingURLs,
			)
		}
	}

	playlist, err := hls.ParsePlaylist(snapshots[0].Manifest)
	if err != nil {
		log.Fatal(err)
	}

	stitched, err := ssai.Stitch(ssai.StitchRequest{
		Playlist:        playlist,
		ReplaceSequence: 341361,
		AdSegmentURL:    "http://localhost:8080/ads/ad-001.ts",
		SourceBaseURL:   "https://aegis-cloudfront-1.tubi.video/c2e3094d-ad56-4c5f-9655-cd80df71fbab/",
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("\nStitched manifest:\n" + stitched)
}
