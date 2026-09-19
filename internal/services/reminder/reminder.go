package reminder

import (
	"context"
	"log/slog"
	"time"

	"github.com/CodeMeAPixel/Mellow/internal/db"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
)

type Service struct {
	client *bot.Client
	store  *db.Store
	every  time.Duration
}

func New(client *bot.Client, store *db.Store) *Service {
	return &Service{client: client, store: store, every: 5 * time.Minute}
}

func (s *Service) Run(ctx context.Context) {
	ticker := time.NewTicker(s.every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *Service) tick(ctx context.Context) {
	due, err := s.store.DueForReminder(ctx)
	if err != nil {
		slog.Warn("reminder query failed", slog.String("err", err.Error()))
		return
	}
	for _, p := range due {
		if p.ReminderMethod != nil && *p.ReminderMethod != "dm" {
			continue
		}
		if db.InQuietHours(p, time.Now()) {
			continue
		}
		if s.dm(ctx, p.ID) {
			interval := p.CheckInInterval
			if interval <= 0 {
				interval = 720
			}
			next := time.Now().Add(time.Duration(interval) * time.Minute)
			now := time.Now()
			if err := s.store.MarkReminderSent(ctx, p.ID, now, &next); err != nil {
				slog.Warn("mark reminder failed", slog.String("err", err.Error()))
			}
		}
	}
}

func (s *Service) dm(ctx context.Context, userID int64) bool {
	ch, err := s.client.Rest.CreateDMChannel(snowflake.ID(userID))
	if err != nil {
		return false
	}
	_, err = s.client.Rest.CreateMessage(ch.ID(), discord.MessageCreate{
		Content: "Gentle reminder to check in with Mellow when you have a moment. Use /checkin to log how you are feeling.",
	})
	return err == nil
}
