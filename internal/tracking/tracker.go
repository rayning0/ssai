// Package tracking implements a deliberately small server-side ad-tracking
// model. It is a prototype: it records media delivery, not proven attention.
package tracking

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type EventType string

const (
	Start         EventType = "start"
	FirstQuartile EventType = "firstQuartile"
	Midpoint      EventType = "midpoint"
	ThirdQuartile EventType = "thirdQuartile"
	Complete      EventType = "complete"
)

type Event struct {
	SessionID    string
	AdInstanceID string
	Type         EventType
	OccurredAt   time.Time
}

type SegmentRequest struct {
	SessionID       string
	AdInstanceID    string
	SegmentStart    time.Duration
	SegmentDuration time.Duration
	AdDuration      time.Duration
}

// Sender represents an external pixel endpoint. Production code would make an
// HTTP request here with deadlines, retry policy, and a dead-letter strategy.
type Sender interface {
	Send(context.Context, Event) error
}

type trackerState struct {
	deliveredThrough time.Duration
	emitted          map[EventType]struct{}
}

type Tracker struct {
	mu     sync.Mutex
	states map[string]*trackerState
	queue  chan Event
	sender Sender
	done   chan struct{}
	wg     sync.WaitGroup
}

func New(sender Sender, queueSize int) *Tracker {
	if queueSize < 1 {
		queueSize = 1
	}
	t := &Tracker{states: make(map[string]*trackerState), queue: make(chan Event, queueSize), sender: sender, done: make(chan struct{})}
	t.wg.Add(1)
	go t.run()
	return t
}

func (t *Tracker) Close() { close(t.done); t.wg.Wait() }

// Observe records delivered media. It is intentionally non-blocking: queue
// pressure drops tracking events rather than delaying video delivery.
func (t *Tracker) Observe(request SegmentRequest) error {
	if request.SessionID == "" || request.AdInstanceID == "" {
		return fmt.Errorf("missing session or ad instance ID")
	}
	if request.AdDuration <= 0 || request.SegmentDuration <= 0 || request.SegmentStart < 0 {
		return fmt.Errorf("invalid media duration")
	}
	through := request.SegmentStart + request.SegmentDuration
	if through > request.AdDuration {
		through = request.AdDuration
	}
	key := request.SessionID + "\x00" + request.AdInstanceID

	t.mu.Lock()
	state := t.states[key]
	if state == nil {
		state = &trackerState{emitted: make(map[EventType]struct{})}
		t.states[key] = state
	}
	if through > state.deliveredThrough {
		state.deliveredThrough = through
	}
	progress := state.deliveredThrough
	for _, milestone := range []struct {
		event EventType
		at    float64
	}{{Start, 0}, {FirstQuartile, .25}, {Midpoint, .5}, {ThirdQuartile, .75}, {Complete, 1}} {
		if progress < time.Duration(milestone.at*float64(request.AdDuration)) {
			continue
		}
		if _, exists := state.emitted[milestone.event]; exists {
			continue
		}
		state.emitted[milestone.event] = struct{}{}
		event := Event{SessionID: request.SessionID, AdInstanceID: request.AdInstanceID, Type: milestone.event, OccurredAt: time.Now()}
		select {
		case t.queue <- event:
		default:
		}
	}
	t.mu.Unlock()
	return nil
}

func (t *Tracker) run() {
	defer t.wg.Done()
	for {
		select {
		case <-t.done:
			return
		case event := <-t.queue:
			if t.sender != nil {
				_ = t.sender.Send(context.Background(), event)
			}
		}
	}
}
