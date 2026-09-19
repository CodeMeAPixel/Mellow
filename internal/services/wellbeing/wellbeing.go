package wellbeing

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/CodeMeAPixel/Mellow/internal/db"
	"github.com/CodeMeAPixel/Mellow/internal/db/gen"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
)

const (
	recapEvery  = 7 * 24 * time.Hour
	promptEvery = 22 * time.Hour
	earliest    = 9
	latest      = 21
)

var prompts = []string{
	"What is one small thing that went okay today?",
	"How is your body feeling right now? A quick scan from head to toes can help.",
	"Is there something you have been putting off that you could make a little smaller?",
	"Who is someone you are glad to have in your life?",
	"What is one thing you can let go of tonight?",
	"What would you say to a friend who felt the way you do today?",
	"Name one thing you can see, hear, and feel right now.",
	"What is something you are looking forward to, even if it is small?",
	"What is one kind thing you could do for yourself today?",
	"When did you feel most like yourself this week?",
	"What is taking up the most space in your mind, and does it need to be there right now?",
	"What is one thing you handled better than you expected lately?",
	"Is there a boundary that would make this week feel easier?",
	"What helped you get through a hard moment recently?",
}

type Service struct {
	client       *bot.Client
	store        *db.Store
	dashboardURL string
	serverPlus   func(ctx context.Context, guildID int64) bool
	every        time.Duration
}

func New(client *bot.Client, store *db.Store, dashboardURL string) *Service {
	return &Service{client: client, store: store, dashboardURL: dashboardURL, every: 20 * time.Minute}
}

func (s *Service) Run(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	case <-time.After(time.Minute):
	}
	ticker := time.NewTicker(s.every)
	defer ticker.Stop()
	for {
		now := time.Now()
		s.tick(ctx, now)
		s.guildPrompts(ctx, now)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Service) tick(ctx context.Context, now time.Time) {
	users, err := s.store.EngagementUsers(ctx)
	if err != nil {
		slog.Warn("wellbeing query failed", slog.String("err", err.Error()))
		return
	}
	for _, p := range users {
		if ctx.Err() != nil {
			return
		}
		if db.InQuietHours(p, now) {
			continue
		}
		if h := now.In(db.UserLocation(p)).Hour(); h < earliest || h >= latest {
			continue
		}
		if s.maybeRecap(ctx, p, now) {
			continue
		}
		s.maybePrompt(ctx, p, now)
	}
}

func due(last *time.Time, every time.Duration, now time.Time) bool {
	return last == nil || now.Sub(*last) >= every
}

func (s *Service) maybeRecap(ctx context.Context, p gen.UserPreferences, now time.Time) bool {
	if !p.WeeklyRecap || !due(p.LastRecapAt, recapEvery, now) {
		return false
	}
	rows, err := s.store.MoodCheckInsSince(ctx, p.ID, now.Add(-recapEvery))
	if err != nil {
		return false
	}
	if err := s.store.MarkRecapSent(ctx, p.ID, now); err != nil {
		return false
	}
	if len(rows) == 0 {
		return false
	}
	s.dm(p.ID, BuildRecap(rows, s.dashboardURL))
	return true
}

func (s *Service) maybePrompt(ctx context.Context, p gen.UserPreferences, now time.Time) {
	if !p.DailyPrompt || !due(p.LastPromptAt, promptEvery, now) {
		return
	}
	if err := s.store.MarkPromptSent(ctx, p.ID, now); err != nil {
		return
	}
	s.dm(p.ID, PromptFor(now))
}

func (s *Service) dm(userID int64, content string) {
	ch, err := s.client.Rest.CreateDMChannel(snowflake.ID(userID))
	if err != nil {
		return
	}
	_, _ = s.client.Rest.CreateMessage(ch.ID(), discord.MessageCreate{Content: content})
}

func PromptFor(t time.Time) string {
	p := prompts[t.YearDay()%len(prompts)]
	return "A gentle prompt for today:\n\n**" + p + "**\n\nReply here if you would like to talk it through. " +
		"Turn these off any time with `/preferences set daily_prompt:false`."
}

func BuildRecap(rows []gen.MoodCheckIn, dashboardURL string) string {
	counts := map[string]int{}
	var sum, n int32
	for _, r := range rows {
		counts[r.Mood]++
		if r.Intensity != nil {
			sum += *r.Intensity
			n++
		}
	}
	type kv struct {
		mood string
		n    int
	}
	ranked := make([]kv, 0, len(counts))
	for m, c := range counts {
		ranked = append(ranked, kv{m, c})
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].n != ranked[j].n {
			return ranked[i].n > ranked[j].n
		}
		return ranked[i].mood < ranked[j].mood
	})
	if len(ranked) > 3 {
		ranked = ranked[:3]
	}
	parts := make([]string, 0, len(ranked))
	for _, r := range ranked {
		parts = append(parts, fmt.Sprintf("%s (%d)", r.mood, r.n))
	}

	var b strings.Builder
	b.WriteString("**Your week with Mellow**\n\n")
	if len(rows) == 1 {
		b.WriteString("You checked in once this week.\n")
	} else {
		fmt.Fprintf(&b, "You checked in %d times this week.\n", len(rows))
	}
	fmt.Fprintf(&b, "Most common feelings: %s.\n", strings.Join(parts, ", "))
	if n > 0 {
		fmt.Fprintf(&b, "Average intensity: %.1f out of 10.\n", float64(sum)/float64(n))
	}
	b.WriteString("\nBe gentle with yourself, whatever the week looked like.")
	if dashboardURL != "" {
		b.WriteString(" See more on your dashboard: " + dashboardURL)
	}
	b.WriteString("\n\nTurn this off any time with `/preferences set weekly_recap:false`.")
	return b.String()
}
