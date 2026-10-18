package hls

type Segment struct {
	Sequence int64
	URI      string
	Duration float64

	InAdBreak bool
	AdElapsed float64
}

type AdBreak struct {
	Duration      float64
	SCTE35        string
	FirstSequence int64
	Active        bool
}

type Playlist struct {
	MediaSequence  int64
	TargetDuration int
	Segments       []Segment
	AdBreak        *AdBreak
}
