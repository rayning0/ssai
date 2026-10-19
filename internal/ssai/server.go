package ssai

import (
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/rayning0/ssai/internal/hls"
)

// Server serves a viewer-facing HLS playlist. Updates are supplied by either a
// fixture loader or a live poller; handlers never fetch from the origin.
type Server struct {
	AdSegmentURL string
	AdMediaPath  string
	SourceURL    string

	mu        sync.RWMutex
	playlist  *hls.Playlist
	manifest  string
	decisions map[int64]struct{}
}

func NewServer(sourceURL, adSegmentURL, adMediaPath string) *Server {
	return &Server{
		SourceURL: sourceURL, AdSegmentURL: adSegmentURL, AdMediaPath: adMediaPath,
		decisions: make(map[int64]struct{}),
	}
}

// UpdatePlaylist atomically publishes a newly fetched source playlist. A
// selected source sequence stays selected for its lifetime, so overlapping
// live playlist responses cannot disagree about which segment is the ad.
func (s *Server) UpdatePlaylist(manifest string, playlist *hls.Playlist) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.playlist, s.manifest = playlist, manifest

	// Preserve a decision while its segment remains in the live window. When it
	// slides out, a later recurring ad opportunity can receive a fresh decision.
	for _, segment := range playlist.Segments {
		if _, selected := s.decisions[segment.Sequence]; selected {
			return
		}
	}
	// One six-second creative per observed ad break is deliberate for this
	// prototype. Avoid short splice-boundary segments when possible.
	for _, segment := range playlist.Segments {
		if segment.InAdBreak && segment.Duration >= 5.5 {
			s.decisions[segment.Sequence] = struct{}{}
			return
		}
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /live/wcvb.m3u8", s.handlePlaylist)
	mux.HandleFunc("GET /ads/ad-001.ts", s.handleAdSegment)
	return mux
}

func (s *Server) handlePlaylist(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.playlist == nil {
		http.Error(w, "source playlist is not available yet", http.StatusServiceUnavailable)
		return
	}
	selected := int64(-1)
	for _, segment := range s.playlist.Segments {
		if _, ok := s.decisions[segment.Sequence]; ok {
			selected = segment.Sequence
			break
		}
	}
	manifest, err := StitchManifest(s.manifest, s.playlist, selected, s.AdSegmentURL, s.SourceURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Cache-Control", "no-store")
	fmt.Fprint(w, manifest)
}

func (s *Server) handleAdSegment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "video/mp2t")
	http.ServeFile(w, r, s.AdMediaPath)
}

func CheckAdMedia(path string) error { _, err := os.Stat(path); return err }
