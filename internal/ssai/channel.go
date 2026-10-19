package ssai

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"

	"github.com/rayning0/ssai/internal/hls"
)

type BreakState struct {
	ID       string
	Duration float64
	SCTE35   string

	// First ad segment observed by this service.
	// May not be the actual beginning of the break.
	FirstObservedSequence int64

	// Largest elapsed time observed at the start of an
	// ad segment, plus that segment's duration.
	ObservedThrough float64

	// True once we've observed a CUE-IN.
	Ended bool
}

func (b *BreakState) Remaining() float64 {
	remaining := b.Duration - b.ObservedThrough
	if remaining < 0 {
		return 0
	}
	return remaining
}

type ChannelState struct {
	ChannelID string

	// Keyed by source media sequence.
	Segments map[int64]hls.Segment

	// Keyed by stable break ID.
	AdBreaks map[string]*BreakState

	LastSequence int64
}

func NewChannelState(channelID string) *ChannelState {
	return &ChannelState{
		ChannelID:    channelID,
		Segments:     make(map[int64]hls.Segment),
		AdBreaks:     make(map[string]*BreakState),
		LastSequence: -1,
	}
}

// BreakID is a deliberately simplified prototype identity.
//
// A production implementation must distinguish separate
// occurrences of a recurring SCTE-35 payload.
func (s *ChannelState) BreakID(scte35 string) string {
	sum := sha256.Sum256([]byte(s.ChannelID + "\x00" + scte35))
	return hex.EncodeToString(sum[:8])
}

func (s *ChannelState) Ingest(p *hls.Playlist) error {
	if p == nil {
		return fmt.Errorf("nil playlist")
	}

	var currentBreak *BreakState

	if p.AdBreak != nil {
		if p.AdBreak.SCTE35 == "" {
			return fmt.Errorf("ad break missing SCTE-35 payload")
		}

		id := s.BreakID(p.AdBreak.SCTE35)

		currentBreak = s.AdBreaks[id]

		if currentBreak == nil {
			currentBreak = &BreakState{
				ID:                    id,
				Duration:              p.AdBreak.Duration,
				SCTE35:                p.AdBreak.SCTE35,
				FirstObservedSequence: -1,
			}
			s.AdBreaks[id] = currentBreak
		}
	}

	for _, segment := range p.Segments {
		if segment.InAdBreak && currentBreak != nil {
			segment.BreakID = currentBreak.ID
		}

		if existing, ok := s.Segments[segment.Sequence]; ok {
			// An overlapping playlist should not silently
			// redefine an already-observed source segment.
			if existing.URI != segment.URI ||
				existing.Duration != segment.Duration ||
				existing.InAdBreak != segment.InAdBreak {
				return fmt.Errorf(
					"conflicting segment at sequence %d",
					segment.Sequence,
				)
			}
		} else {
			s.Segments[segment.Sequence] = segment
		}

		if segment.Sequence > s.LastSequence {
			s.LastSequence = segment.Sequence
		}

		if !segment.InAdBreak || currentBreak == nil {
			continue
		}

		if currentBreak.FirstObservedSequence == -1 ||
			segment.Sequence < currentBreak.FirstObservedSequence {
			currentBreak.FirstObservedSequence = segment.Sequence
		}

		through := segment.AdElapsed + segment.Duration
		if through > currentBreak.ObservedThrough {
			currentBreak.ObservedThrough = through
		}
	}

	if currentBreak != nil && !p.AdBreak.Active {
		currentBreak.Ended = true
	}

	return nil
}

func (s *ChannelState) SortedSegments() []hls.Segment {
	result := make([]hls.Segment, 0, len(s.Segments))

	for _, segment := range s.Segments {
		result = append(result, segment)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Sequence < result[j].Sequence
	})

	return result
}
