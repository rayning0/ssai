package hls

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
)

func ParsePlaylist(manifest string) (*Playlist, error) {
	p := &Playlist{}

	scanner := bufio.NewScanner(strings.NewReader(manifest))

	var (
		pendingDuration float64
		hasDuration     bool
		inAdBreak       bool
		adElapsed       float64
		scte35          string
	)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		switch {
		case strings.HasPrefix(line, "#EXT-X-MEDIA-SEQUENCE:"):
			value := strings.TrimPrefix(line, "#EXT-X-MEDIA-SEQUENCE:")
			n, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid media sequence: %w", err)
			}
			p.MediaSequence = n

		case strings.HasPrefix(line, "#EXT-X-TARGETDURATION:"):
			value := strings.TrimPrefix(line, "#EXT-X-TARGETDURATION:")
			n, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("invalid target duration: %w", err)
			}
			p.TargetDuration = n

		case strings.HasPrefix(line, "#EXT-OATCLS-SCTE35:"):
			scte35 = strings.TrimPrefix(line, "#EXT-OATCLS-SCTE35:")

		case strings.HasPrefix(line, "#EXT-X-CUE-OUT:"):
			value := strings.TrimPrefix(line, "#EXT-X-CUE-OUT:")
			duration, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid cue-out duration: %w", err)
			}

			inAdBreak = true
			adElapsed = 0

			p.AdBreak = &AdBreak{
				Duration:      duration,
				SCTE35:        scte35,
				FirstSequence: p.MediaSequence + int64(len(p.Segments)),
				Active:        true,
			}

		case strings.HasPrefix(line, "#EXT-X-CUE-OUT-CONT:"):
			value := strings.TrimPrefix(line, "#EXT-X-CUE-OUT-CONT:")

			attrs := parseAttributes(value)

			elapsed, err := strconv.ParseFloat(attrs["ElapsedTime"], 64)
			if err != nil {
				return nil, fmt.Errorf("invalid elapsed time: %w", err)
			}

			duration, err := strconv.ParseFloat(attrs["Duration"], 64)
			if err != nil {
				return nil, fmt.Errorf("invalid break duration: %w", err)
			}

			inAdBreak = true
			adElapsed = elapsed

			if p.AdBreak == nil {
				p.AdBreak = &AdBreak{
					Duration:      duration,
					SCTE35:        attrs["SCTE35"],
					FirstSequence: p.MediaSequence + int64(len(p.Segments)),
					Active:        true,
				}
			}

		case line == "#EXT-X-CUE-IN":
			inAdBreak = false

			if p.AdBreak != nil {
				p.AdBreak.Active = false
			}

		case strings.HasPrefix(line, "#EXTINF:"):
			value := strings.TrimPrefix(line, "#EXTINF:")
			value = strings.SplitN(value, ",", 2)[0]

			duration, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid segment duration: %w", err)
			}

			pendingDuration = duration
			hasDuration = true

		case line != "" && !strings.HasPrefix(line, "#"):
			if !hasDuration {
				return nil, fmt.Errorf("segment without EXTINF: %s", line)
			}

			p.Segments = append(p.Segments, Segment{
				Sequence:  p.MediaSequence + int64(len(p.Segments)),
				URI:       line,
				Duration:  pendingDuration,
				InAdBreak: inAdBreak,
				AdElapsed: adElapsed,
			})

			hasDuration = false
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return p, nil
}

func parseAttributes(s string) map[string]string {
	result := make(map[string]string)

	for _, part := range strings.Split(s, ",") {
		key, value, ok := strings.Cut(part, "=")
		if ok {
			result[key] = value
		}
	}

	return result
}
