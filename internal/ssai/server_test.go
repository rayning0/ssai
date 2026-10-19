package ssai

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rayning0/ssai/internal/hls"
)

const sourceURL = "https://origin.example/live/playlist.m3u8"

func TestServerSelectsNewBreakAfterOldDecisionSlidesOut(t *testing.T) {
	first := "#EXTM3U\n#EXT-X-TARGETDURATION:6\n#EXT-X-MEDIA-SEQUENCE:100\n#EXT-OATCLS-SCTE35:signal\n#EXT-X-CUE-OUT:30\n#EXTINF:6,\na.ts\n#EXT-X-CUE-OUT-CONT:ElapsedTime=6,Duration=30,SCTE35=signal\n#EXTINF:6,\nb.ts\n"
	second := "#EXTM3U\n#EXT-X-TARGETDURATION:6\n#EXT-X-MEDIA-SEQUENCE:101\n#EXT-X-CUE-OUT-CONT:ElapsedTime=6,Duration=30,SCTE35=signal\n#EXTINF:6,\nb.ts\n#EXT-X-CUE-OUT-CONT:ElapsedTime=12,Duration=30,SCTE35=signal\n#EXTINF:6,\nc.ts\n"
	server := NewServer(sourceURL, "http://localhost:8080/ads/ad-001.ts", "testdata/ads/ad-001.ts")
	playlist, err := hls.ParsePlaylist(first)
	if err != nil {
		t.Fatal(err)
	}
	server.UpdatePlaylist(first, playlist)
	playlist, err = hls.ParsePlaylist(second)
	if err != nil {
		t.Fatal(err)
	}
	server.UpdatePlaylist(second, playlist)

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/live/wcvb.m3u8", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "ad-001.ts") {
		t.Fatal("new visible ad opportunity was not replaced")
	}
	if !strings.Contains(recorder.Body.String(), "https://origin.example/live/c.ts") {
		t.Fatal("source media URI was not resolved against the origin")
	}
}

func TestStitchManifestInsertsAdAndPreservesSourceTags(t *testing.T) {
	manifest := "#EXTM3U\n#EXT-X-TARGETDURATION:6\n#EXT-X-MEDIA-SEQUENCE:100\n#EXT-X-CUE-OUT:30\n#EXTINF:6,\na.ts\n"
	playlist, err := hls.ParsePlaylist(manifest)
	if err != nil {
		t.Fatal(err)
	}
	got, err := StitchManifest(manifest, playlist, 100, "http://localhost:8080/ads/ad-001.ts", sourceURL)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "#EXT-X-CUE-OUT:30") || !strings.Contains(got, "#EXT-X-DISCONTINUITY\n#EXTINF:6,\nhttp://localhost:8080/ads/ad-001.ts") {
		t.Fatalf("unexpected stitched manifest:\n%s", got)
	}
}
