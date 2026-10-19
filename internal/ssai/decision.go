package ssai

import (
	"fmt"
	"time"
)

type AdRequest struct {
	SessionID         string
	BreakID           string
	RemainingDuration time.Duration
}

type AdDecision struct {
	Ads []SelectedAd
}

type MockAdServer struct {
	Ads []SelectedAd
}

func (s *MockAdServer) Decide(req AdRequest) (AdDecision, error) {
	if req.SessionID == "" || req.BreakID == "" {
		return AdDecision{}, fmt.Errorf("missing session or break ID")
	}

	// Prototype policy: don't initiate a new decision when
	// fewer than six seconds remain in the break.
	if req.RemainingDuration < 6*time.Second {
		return AdDecision{}, nil
	}

	for _, ad := range s.Ads {
		if ad.Duration <= req.RemainingDuration {
			return AdDecision{
				Ads: []SelectedAd{ad},
			}, nil
		}
	}

	// No suitable ad: the stitcher will retain source media.
	return AdDecision{}, nil
}
