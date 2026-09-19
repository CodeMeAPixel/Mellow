package db

import (
	"context"
	"sort"
	"time"

	"github.com/CodeMeAPixel/Mellow/internal/db/gen"
)

func (s *Store) RecordCopingToolUsageIn(ctx context.Context, userID int64, tool string, guildID *int64) error {
	_, err := s.q.RecordCopingToolUsageInGuild(ctx, gen.RecordCopingToolUsageInGuildParams{UserId: userID, ToolName: tool, GuildId: guildID})
	return norm(err)
}

func (s *Store) SetGuildSchedule(ctx context.Context, guildID int64, enabled bool, day *int32, hour int32, tz *string) error {
	return norm(s.q.SetGuildSchedule(ctx, gen.SetGuildScheduleParams{
		ID: guildID, PromptEnabled: enabled, PromptDay: day, PromptHour: hour, PromptTimezone: tz,
	}))
}

func (s *Store) SetExtraAlertChannels(ctx context.Context, guildID int64, ids []string) error {
	if ids == nil {
		ids = []string{}
	}
	return norm(s.q.SetExtraAlertChannels(ctx, gen.SetExtraAlertChannelsParams{ID: guildID, ExtraAlertChannelIds: ids}))
}

func (s *Store) GuildsWithPrompts(ctx context.Context) ([]gen.Guild, error) {
	rows, err := s.q.GuildsWithPrompts(ctx)
	return rows, norm(err)
}

func (s *Store) MarkGuildPromptSent(ctx context.Context, guildID int64, at time.Time) error {
	return norm(s.q.MarkGuildPromptSent(ctx, gen.MarkGuildPromptSentParams{ID: guildID, LastPromptAt: &at}))
}

type ActivityWeek struct {
	Week     time.Time `json:"week"`
	CheckIns int64     `json:"checkIns"`
	ToolUses int64     `json:"toolUses"`
}

// GuildActivity returns weekly totals of check-ins and coping tool uses made
// in a server. These are event counts only: no users, moods, or content.
func (s *Store) GuildActivity(ctx context.Context, guildID int64, weeks int) ([]ActivityWeek, error) {
	since := time.Now().AddDate(0, 0, -7*weeks)
	checks, err := s.q.GuildWeeklyCheckIns(ctx, gen.GuildWeeklyCheckInsParams{GuildId: &guildID, CreatedAt: since})
	if err != nil {
		return nil, norm(err)
	}
	tools, err := s.q.GuildWeeklyToolUses(ctx, gen.GuildWeeklyToolUsesParams{GuildId: &guildID, UsedAt: since})
	if err != nil {
		return nil, norm(err)
	}

	byWeek := map[time.Time]*ActivityWeek{}
	get := func(t time.Time) *ActivityWeek {
		if w, ok := byWeek[t]; ok {
			return w
		}
		w := &ActivityWeek{Week: t}
		byWeek[t] = w
		return w
	}
	for _, r := range checks {
		get(r.Week).CheckIns = r.Total
	}
	for _, r := range tools {
		get(r.Week).ToolUses = r.Total
	}

	out := make([]ActivityWeek, 0, len(byWeek))
	for _, w := range byWeek {
		out = append(out, *w)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Week.Before(out[j].Week) })
	return out, nil
}
