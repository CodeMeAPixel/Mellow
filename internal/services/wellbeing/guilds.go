package wellbeing

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/CodeMeAPixel/Mellow/internal/db/gen"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
)

const guildPromptEvery = 20 * time.Hour

var guildPrompts = []string{
	"How is everyone's energy today? Share a word or an emoji if you like.",
	"What is one small thing that made today a little easier?",
	"Take a breath with us: in for four, hold for four, out for four. How are you feeling after?",
	"Is there something you are looking forward to this week?",
	"What is one kind thing you could do for yourself today?",
	"If your mood had a weather forecast right now, what would it be?",
	"What is something that helped you get through a hard day recently?",
	"Who is someone you are grateful for this week? Tell them, or tell us.",
	"Stretch, sip some water, and unclench your jaw. How is your body feeling?",
	"What is one thing you are proud of lately, however small?",
}

func guildLocation(g gen.Guild) *time.Location {
	if g.PromptTimezone != nil && *g.PromptTimezone != "" {
		if loc, err := time.LoadLocation(*g.PromptTimezone); err == nil {
			return loc
		}
	}
	return time.UTC
}

func GuildPromptDue(g gen.Guild, now time.Time) bool {
	if !g.PromptEnabled {
		return false
	}
	local := now.In(guildLocation(g))
	if local.Hour() != int(g.PromptHour) {
		return false
	}
	if g.PromptDay != nil && int(local.Weekday()) != int(*g.PromptDay) {
		return false
	}
	return due(g.LastPromptAt, guildPromptEvery, now)
}

func GuildPromptText(now time.Time) string {
	p := guildPrompts[now.YearDay()%len(guildPrompts)]
	return "**Community check-in**\n\n" + p +
		"\n\nWant to log how you are feeling? Use `/checkin`. Only you see your check-in. " +
		"For a calm moment, try `/guided`."
}

func (s *Service) WithServerPlus(has func(ctx context.Context, guildID int64) bool) *Service {
	s.serverPlus = has
	return s
}

func (s *Service) guildPrompts(ctx context.Context, now time.Time) {
	if s.serverPlus == nil {
		return
	}
	guilds, err := s.store.GuildsWithPrompts(ctx)
	if err != nil {
		slog.Warn("guild prompt query failed", slog.String("err", err.Error()))
		return
	}
	for _, g := range guilds {
		if ctx.Err() != nil {
			return
		}
		if !GuildPromptDue(g, now) || g.CheckInChannelId == nil || *g.CheckInChannelId == "" {
			continue
		}
		if !s.serverPlus(ctx, g.ID) {
			continue
		}
		if err := s.store.MarkGuildPromptSent(ctx, g.ID, now); err != nil {
			continue
		}
		chID, err := snowflake.Parse(*g.CheckInChannelId)
		if err != nil {
			continue
		}
		if _, err := s.client.Rest.CreateMessage(chID, discord.MessageCreate{
			Content:         GuildPromptText(now),
			AllowedMentions: &discord.AllowedMentions{},
		}); err != nil {
			slog.Warn("guild prompt post failed", slog.String("guild", strconv.FormatInt(g.ID, 10)), slog.String("err", err.Error()))
		}
	}
}
