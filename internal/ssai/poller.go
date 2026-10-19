package ssai

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rayning0/ssai/internal/hls"
)

// Poller refreshes a live origin independently of player requests. A failed
// refresh leaves the last known-good playlist available for playback.
type Poller struct {
	URL      string
	Interval time.Duration
	Client   *http.Client
	Server   *Server
}

func (p *Poller) Run(ctx context.Context) error {
	if p.Interval <= 0 {
		return fmt.Errorf("poll interval must be positive")
	}
	ticker := time.NewTicker(p.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			_ = p.PollOnce(ctx)
		}
	}
}

func (p *Poller) PollOnce(ctx context.Context) error {
	client := p.Client
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.URL, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch source playlist: %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}
	playlist, err := hls.ParsePlaylist(string(body))
	if err != nil {
		return fmt.Errorf("parse source playlist: %w", err)
	}
	p.Server.UpdatePlaylist(string(body), playlist)
	return nil
}
