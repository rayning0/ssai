package ssai

import (
	"fmt"
	"strings"

	"github.com/rayning0/ssai/internal/hls"
)

type StitchRequest struct {
	Playlist        *hls.Playlist
	ReplaceSequence int64
	AdSegmentURL    string
	SourceBaseURL   string
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
