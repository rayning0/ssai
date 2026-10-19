package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/rayning0/ssai/internal/hls"
	"github.com/rayning0/ssai/internal/ssai"
	"github.com/rayning0/ssai/internal/tracking"
)

const liveWCVBURL = "https://aegis-cloudfront-1.tubi.video/c2e3094d-ad56-4c5f-9655-cd80df71fbab/playlist_1280x720.m3u8"

func main() {
	mode := flag.String("mode", "fixture", "playlist source: fixture or live")
	fixture := flag.String("fixture", "testdata/sample-scte-playlist.txt", "snapshot fixture path")
	sourceURL := flag.String("source-url", liveWCVBURL, "live HLS media-playlist URL")
	pollInterval := flag.Duration("poll-interval", 3*time.Second, "live origin refresh interval")
	addr := flag.String("addr", ":8080", "HTTP listen address")
	flag.Parse()

	if err := ssai.CheckAdMedia("testdata/ad/ad000.ts"); err != nil {
		log.Fatal("ad media missing: ", err)
	}
	pixelTracker := tracking.New(logPixelSender{}, 100)
	defer pixelTracker.Close()
	server := ssai.NewServer(*sourceURL, "http://localhost:8080/ads/ad-001.ts?session=demo-viewer&ad_instance=demo-ad-001", "testdata/ad/ad000.ts")
	server.Tracker = pixelTracker
	switch *mode {
	case "fixture":
		if err := loadFixture(server, *fixture); err != nil {
			log.Fatal(err)
		}
	case "live":
		poller := &ssai.Poller{URL: *sourceURL, Interval: *pollInterval, Client: &http.Client{Timeout: 10 * time.Second}, Server: server}
		if err := poller.PollOnce(context.Background()); err != nil {
			log.Fatal("initial live playlist fetch: ", err)
		}
		go func() {
			if err := poller.Run(context.Background()); err != nil && err != context.Canceled {
				log.Printf("live polling stopped: %v", err)
			}
		}()
	default:
		log.Fatalf("invalid -mode %q (want fixture or live)", *mode)
	}
	fmt.Printf("SSAI server (%s mode) at http://localhost%s/live/wcvb.m3u8\n", *mode, *addr)
	log.Fatal(http.ListenAndServe(*addr, server.Handler()))
}

type logPixelSender struct{}

func (logPixelSender) Send(_ context.Context, event tracking.Event) error {
	log.Printf("pixel sent session=%s ad_instance=%s event=%s", event.SessionID, event.AdInstanceID, event.Type)
	return nil
}

func loadFixture(server *ssai.Server, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	snapshots, err := hls.ReadSnapshots(file)
	if err != nil {
		return err
	}
	if len(snapshots) == 0 {
		return fmt.Errorf("fixture has no HLS snapshots")
	}
	playlist, err := hls.ParsePlaylist(snapshots[0].Manifest)
	if err != nil {
		return fmt.Errorf("parse fixture: %w", err)
	}
	server.UpdatePlaylist(snapshots[0].Manifest, playlist)
	return nil
}
