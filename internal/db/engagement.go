package db

import (
	"context"
	"time"

	"github.com/CodeMeAPixel/Mellow/internal/db/gen"
)

func (s *Store) SetQuietHours(ctx context.Context, userID int64, start, end *int32) error {
	if _, err := s.q.EnsureUserPreferences(ctx, userID); err != nil {
		return norm(err)
	}
	return norm(s.q.SetQuietHours(ctx, gen.SetQuietHoursParams{ID: userID, QuietStart: start, QuietEnd: end}))
}

func (s *Store) MarkRecapSent(ctx context.Context, userID int64, at time.Time) error {
	return norm(s.q.MarkRecapSent(ctx, gen.MarkRecapSentParams{ID: userID, LastRecapAt: &at}))
}

func (s *Store) MarkPromptSent(ctx context.Context, userID int64, at time.Time) error {
	return norm(s.q.MarkPromptSent(ctx, gen.MarkPromptSentParams{ID: userID, LastPromptAt: &at}))
}

func (s *Store) EngagementUsers(ctx context.Context) ([]gen.UserPreferences, error) {
	rows, err := s.q.EngagementUsers(ctx)
	return rows, norm(err)
}

func UserLocation(p gen.UserPreferences) *time.Location {
	if p.Timezone != nil && *p.Timezone != "" {
		if loc, err := time.LoadLocation(*p.Timezone); err == nil {
			return loc
		}
	}
	return time.UTC
}

func InQuietHours(p gen.UserPreferences, now time.Time) bool {
	if p.QuietStart == nil || p.QuietEnd == nil || *p.QuietStart == *p.QuietEnd {
		return false
	}
	h := int32(now.In(UserLocation(p)).Hour())
	start, end := *p.QuietStart, *p.QuietEnd
	if start < end {
		return h >= start && h < end
	}
	return h >= start || h < end
}
