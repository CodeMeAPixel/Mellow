package statusposter

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

type Service struct {
	url     string
	key     string
	version string
	shards  func() int
	guilds  func() int
	users   func() int
	client  *http.Client
	every   time.Duration
}

func New(url, key, version string, shards, guilds, users func() int) *Service {
	if shards == nil {
		shards = func() int { return 1 }
	}
	if guilds == nil {
		guilds = func() int { return 0 }
	}
	if users == nil {
		users = func() int { return 0 }
	}
	return &Service{
		url:     url,
		key:     key,
		version: version,
		shards:  shards,
		guilds:  guilds,
		users:   users,
		client:  &http.Client{Timeout: 20 * time.Second},
		every:   5 * time.Minute,
	}
}

func (s *Service) Run(ctx context.Context) {
	if s.key == "" || s.url == "" {
		slog.Info("status poster disabled (no MELLOW_STATUS_API_KEY)")
		return
	}
	select {
	case <-ctx.Done():
		return
	case <-time.After(30 * time.Second):
	}
	s.post(ctx)
	t := time.NewTicker(s.every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.post(ctx)
		}
	}
}

type payload struct {
	Status     bool   `json:"status"`
	ShardCount int    `json:"shardCount"`
	GuildCount int    `json:"guildCount"`
	UserCount  int    `json:"userCount"`
	Version    string `json:"version"`
	Message    string `json:"message"`
	Timestamp  string `json:"timestamp"`
	APIKey     string `json:"apiKey"`
}

func (s *Service) post(ctx context.Context) {
	body, _ := json.Marshal(payload{
		Status:     true,
		ShardCount: s.shards(),
		GuildCount: s.guilds(),
		UserCount:  s.users(),
		Version:    s.version,
		Message:    "All systems operational",
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		APIKey:     s.key,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.url, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "MellowBot/"+s.version)

	resp, err := s.client.Do(req)
	if err != nil {
		slog.Warn("status post failed", slog.String("err", err.Error()))
		return
	}
	resp.Body.Close()
	if resp.StatusCode >= 300 {
		slog.Warn("status post rejected", slog.Int("code", resp.StatusCode))
	}
}
