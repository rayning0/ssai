package tracking

import (
	"context"
	"sync"
	"testing"
	"time"
)

type recordingSender struct {
	mu     sync.Mutex
	events []Event
}

func (s *recordingSender) Send(_ context.Context, event Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	return nil
}

func TestObserveEmitsEachMilestoneOnce(t *testing.T) {
	sender := &recordingSender{}
	tracker := New(sender, 10)
	defer tracker.Close()
	request := SegmentRequest{SessionID: "viewer", AdInstanceID: "ad-instance", AdDuration: 8 * time.Second, SegmentDuration: 2 * time.Second}
	for start := time.Duration(0); start < 8*time.Second; start += 2 * time.Second {
		request.SegmentStart = start
		if err := tracker.Observe(request); err != nil {
			t.Fatal(err)
		}
	}
	time.Sleep(20 * time.Millisecond)
	sender.mu.Lock()
	defer sender.mu.Unlock()
	if len(sender.events) != 5 {
		t.Fatalf("events = %d, want 5", len(sender.events))
	}
	if sender.events[0].Type != Start || sender.events[4].Type != Complete {
		t.Fatalf("unexpected event sequence: %#v", sender.events)
	}
}
