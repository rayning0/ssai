// fake-origin is a local HLS origin that loops five program fragments followed
// by two ad fragments. Its URLs are virtual media-sequence URLs, mapped to the
// fixed files on disk so media filenames never need to change.
package main

import (
	"flag"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	windowSegments = 10
	scte35         = "/DAlAAAAAAAAAP/wFAUABhOOf+//+KlxBf4BEUiEAAEBAQAAc+HyHQ=="
)

type asset struct {
	path      string
	duration  time.Duration
	isAd      bool
	adElapsed time.Duration
}

var (
	startedAt = time.Now()
	assets    = []asset{
		{"testdata/program/program000.ts", 6 * time.Second, false, 0},
		{"testdata/program/program001.ts", 6 * time.Second, false, 0},
		{"testdata/program/program002.ts", 6 * time.Second, false, 0},
		{"testdata/program/program003.ts", 6 * time.Second, false, 0},
		{"testdata/program/program004.ts", 6 * time.Second, false, 0},
		{"testdata/ad/ad000.ts", 6 * time.Second, true, 0},
		{"testdata/ad/ad001.ts", 4 * time.Second, true, 6 * time.Second},
	}
)

func main() {
	addr := flag.String("addr", ":8090", "HTTP listen address")
	flag.Parse()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /live/fake.m3u8", fakePlaylist)
	mux.HandleFunc("GET /segments/", func(w http.ResponseWriter, r *http.Request) {
		sequence, ok := sequenceFromPath(r.URL.Path)
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "video/mp2t")
		http.ServeFile(w, r, filepath.Clean(assetForSequence(sequence).path))
	})
	fmt.Printf("Fake origin at http://localhost%s/live/fake.m3u8\n", *addr)
	fmt.Println("Loop: 30-second program in five fragments, then 10-second ad in two fragments")
	if err := http.ListenAndServe(*addr, cors(mux)); err != nil {
		panic(err)
	}
}

// The stitched playlist is served by localhost:8080 while source segments are
// served here on localhost:8090. Apply headers before ServeFile so they are
// present on both normal and HTTP Range (206) responses.
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:8080")
		w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Range")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Range, Accept-Ranges")
		w.Header().Set("Vary", "Origin")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func fakePlaylist(w http.ResponseWriter, r *http.Request) {
	nowSequence := currentSequence(time.Since(startedAt))
	start := nowSequence - (windowSegments - 1)
	if start < 0 {
		start = 0
	}
	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Cache-Control", "no-store")
	fmt.Fprintln(w, "#EXTM3U")
	fmt.Fprintln(w, "#EXT-X-VERSION:3")
	fmt.Fprintln(w, "#EXT-X-TARGETDURATION:6")
	fmt.Fprintf(w, "#EXT-X-MEDIA-SEQUENCE:%d\n", start)

	wasAd := false
	for sequence := start; sequence <= nowSequence; sequence++ {
		asset := assetForSequence(sequence)
		if asset.isAd && !wasAd {
			fmt.Fprintln(w, "#EXT-OATCLS-SCTE35:"+scte35)
			fmt.Fprintln(w, "#EXT-X-CUE-OUT:10.000")
		} else if asset.isAd {
			fmt.Fprintf(w, "#EXT-X-CUE-OUT-CONT:ElapsedTime=%.3f,Duration=10.000,SCTE35=%s\n", asset.adElapsed.Seconds(), scte35)
		} else if wasAd {
			fmt.Fprintln(w, "#EXT-X-CUE-IN")
		}
		if sequence != start {
			fmt.Fprintln(w, "#EXT-X-DISCONTINUITY")
		}
		fmt.Fprintf(w, "#EXTINF:%.3f,\n", asset.duration.Seconds())
		fmt.Fprintf(w, "/segments/%d.ts\n", sequence)
		wasAd = asset.isAd
	}
}

func currentSequence(elapsed time.Duration) int64 {
	cycleDuration := 40 * time.Second
	cycle := int64(elapsed / cycleDuration)
	position := elapsed % cycleDuration
	if position < 30*time.Second {
		return cycle*int64(len(assets)) + int64(position/(6*time.Second))
	}
	if position < 36*time.Second {
		return cycle*int64(len(assets)) + 5
	}
	return cycle*int64(len(assets)) + 6
}

func assetForSequence(sequence int64) asset { return assets[sequence%int64(len(assets))] }

func sequenceFromPath(path string) (int64, bool) {
	name := strings.TrimPrefix(path, "/segments/")
	if !strings.HasSuffix(name, ".ts") || strings.Contains(name, "/") {
		return 0, false
	}
	sequence, err := strconv.ParseInt(strings.TrimSuffix(name, ".ts"), 10, 64)
	return sequence, err == nil && sequence >= 0
}
