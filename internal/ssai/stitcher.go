package ssai

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/rayning0/ssai/internal/hls"
)

type StitchRequest struct {
	Playlist        *hls.Playlist
	ReplaceSequence int64
	AdSegmentURL    string
	SourceBaseURL   string
}

// StitchManifest preserves source HLS tags while resolving source media URIs
// to the origin. A negative replaceSequence performs only that URI rewrite,
// which is the safe fallback when no ad break is currently available.
func StitchManifest(manifest string, playlist *hls.Playlist, replaceSequence int64, adURL, sourceURL string) (string, error) {
	if playlist == nil {
		return "", fmt.Errorf("nil playlist")
	}
	base, err := url.Parse(sourceURL)
	if err != nil {
		return "", fmt.Errorf("parse source URL: %w", err)
	}
	var out strings.Builder
	var pending []string
	segmentIndex := 0
	for _, line := range strings.Split(manifest, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			pending = append(pending, line)
			continue
		}
		if segmentIndex >= len(playlist.Segments) {
			return "", fmt.Errorf("more media URIs than parsed segments")
		}
		segment := playlist.Segments[segmentIndex]
		segmentIndex++
		if segment.Sequence == replaceSequence {
			if !segment.InAdBreak {
				return "", fmt.Errorf("sequence %d is not inside an ad break", segment.Sequence)
			}
			extinf := len(pending)
			for i := len(pending) - 1; i >= 0; i-- {
				if strings.HasPrefix(strings.TrimSpace(pending[i]), "#EXTINF:") {
					extinf = i
					break
				}
			}
			for _, pendingLine := range pending[:extinf] {
				fmt.Fprintln(&out, pendingLine)
			}
			fmt.Fprintln(&out, "#EXT-X-DISCONTINUITY")
			for _, pendingLine := range pending[extinf:] {
				fmt.Fprintln(&out, pendingLine)
			}
			fmt.Fprintln(&out, adURL)
			fmt.Fprintln(&out, "#EXT-X-DISCONTINUITY")
		} else {
			for _, pendingLine := range pending {
				fmt.Fprintln(&out, pendingLine)
			}
			uri, err := url.Parse(trimmed)
			if err != nil {
				return "", fmt.Errorf("parse segment URL: %w", err)
			}
			fmt.Fprintln(&out, base.ResolveReference(uri).String())
		}
		pending = pending[:0]
	}
	for _, pendingLine := range pending {
		fmt.Fprintln(&out, pendingLine)
	}
	return out.String(), nil
}

func Stitch(req StitchRequest) (string, error) {
	if req.Playlist == nil {
		return "", fmt.Errorf("nil playlist")
	}

	if req.AdSegmentURL == "" {
		return "", fmt.Errorf("missing ad segment URL")
	}

	var out strings.Builder

	fmt.Fprintln(&out, "#EXTM3U")
	fmt.Fprintln(&out, "#EXT-X-VERSION:3")
	fmt.Fprintf(&out, "#EXT-X-TARGETDURATION:%d\n",
		req.Playlist.TargetDuration)
	fmt.Fprintf(&out, "#EXT-X-MEDIA-SEQUENCE:%d\n",
		req.Playlist.MediaSequence)

	replaced := false

	for i, segment := range req.Playlist.Segments {
		if segment.Sequence == req.ReplaceSequence {
			if !segment.InAdBreak {
				return "", fmt.Errorf(
					"sequence %d is not inside an ad break",
					segment.Sequence,
				)
			}

			fmt.Fprintln(&out, "#EXT-X-DISCONTINUITY")
			fmt.Fprintf(&out, "#EXTINF:%.5f,\n", segment.Duration)
			fmt.Fprintln(&out, req.AdSegmentURL)

			replaced = true

			// Restore source media timeline before the next segment.
			if i+1 < len(req.Playlist.Segments) {
				fmt.Fprintln(&out, "#EXT-X-DISCONTINUITY")
			}

			continue
		}

		fmt.Fprintf(&out, "#EXTINF:%.5f,\n", segment.Duration)

		// The original source segment URI is relative.
		// Resolve it against the source playlist's directory.
		fmt.Fprintln(&out, req.SourceBaseURL+segment.URI)
	}

	if !replaced {
		return "", fmt.Errorf(
			"replacement sequence %d not found",
			req.ReplaceSequence,
		)
	}

	return out.String(), nil
}
