package discord

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/CodeMeAPixel/Mellow/internal/db"
	"github.com/CodeMeAPixel/Mellow/internal/db/gen"
	"github.com/disgoorg/disgo/discord"
)

var moodEmoji = map[string]string{
	"happy": "😊", "calm": "😌", "neutral": "😐", "sad": "😔", "anxious": "😟",
	"frustrated": "😤", "tired": "😴", "confused": "🤔", "angry": "😠", "upset": "😢",
	"overwhelmed": "😩", "numb": "😶", "stressed": "😬", "embarrassed": "😳", "grateful": "😇",
	"confident": "😎", "excited": "🥳", "disappointed": "😕", "afraid": "😱", "lonely": "🥺",
	"proud": "😏", "relieved": "😅", "hopeful": "🌅", "content": "🙂", "motivated": "💪",
}

func moodChoices() []discord.ApplicationCommandOptionChoiceString {
	order := []string{
		"happy", "calm", "neutral", "sad", "anxious", "frustrated", "tired", "confused",
		"angry", "upset", "overwhelmed", "numb", "stressed", "embarrassed", "grateful",
		"confident", "excited", "disappointed", "afraid", "lonely", "proud", "relieved",
		"hopeful", "content", "motivated",
	}
	out := make([]discord.ApplicationCommandOptionChoiceString, 0, len(order))
	for _, m := range order {
		out = append(out, discord.ApplicationCommandOptionChoiceString{
			Name:  moodEmoji[m] + " " + strings.Title(m),
			Value: m,
		})
	}
	return out
}

func checkinCommands() []*Command {
	intensityChoices := []discord.ApplicationCommandOptionChoiceInt{
		{Name: "1 (very low)", Value: 1}, {Name: "2 (low)", Value: 2}, {Name: "3 (moderate)", Value: 3},
		{Name: "4 (high)", Value: 4}, {Name: "5 (very high)", Value: 5},
	}

	return []*Command{
		{
			Name: "checkin", Description: "Log your current mood and reflect on how you are feeling.",
			Category: "Check-In", Cooldown: 15 * time.Second,
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionString{Name: "mood", Description: "How are you feeling right now?", Required: true, Choices: moodChoices()},
				discord.ApplicationCommandOptionInt{Name: "intensity", Description: "How strong is this mood? (1-5)", Choices: intensityChoices},
				discord.ApplicationCommandOptionString{Name: "activity", Description: "What are you doing right now?"},
				discord.ApplicationCommandOptionString{Name: "note", Description: "Anything else you want to add?"},
			},
			Run: runCheckin,
		},
		{
			Name: "insights", Description: "Get insights and trends from your mood check-ins.",
			Category: "Check-In", Cooldown: 10 * time.Second,
			Run: runInsights,
		},
	}
}

func runCheckin(ctx context.Context, c *Ctx) error {
	mood := c.String("mood")
	intensity := int32(3)
	if v, ok := c.Int("intensity"); ok {
		intensity = int32(v)
	}
	var activity, note *string
	if v := c.String("activity"); v != "" {
		activity = &v
	}
	if v := c.String("note"); v != "" {
		note = &v
	}

	prefs, err := c.Store.EnsureUserPreferences(ctx, c.UserID)
	if err != nil {
		return err
	}
	interval := prefs.CheckInInterval
	if interval <= 0 {
		interval = 720
	}

	if last, err := c.Store.LastMoodCheckIn(ctx, c.UserID); err == nil {
		gap := time.Since(last.CreatedAt)
		window := time.Duration(interval) * time.Minute
		if gap < window {
			remaining := (window - gap).Round(time.Minute)
			return c.ReplyEphemeral(fmt.Sprintf(
				"You can check in once every %d minutes. Please wait about %s.", interval, remaining))
		}
	}

	next := time.Now().Add(time.Duration(interval) * time.Minute)
	if _, err := c.Store.CreateMoodCheckIn(ctx, db.NewMoodCheckIn{
		UserID: c.UserID, Mood: mood, Intensity: &intensity,
		Activity: activity, Note: note, NextCheckIn: &next,
	}); err != nil {
		return err
	}
	if err := c.Store.SetNextCheckIn(ctx, c.UserID, &next); err != nil {
		return err
	}
	_ = c.Store.RecordCopingToolUsage(ctx, c.UserID, "checkin")

	recent, _ := c.Store.RecentMoodCheckIns(ctx, c.UserID, 5)
	emb := successEmbed("Check-in complete", checkinSummary(mood, intensity, activity, note)).
		AddField("Recent check-ins", renderRecent(recent), false).
		WithFooter(fmt.Sprintf("Next reminder around %s", next.Format("Jan 2 15:04")), brand().AvatarURL)
	return c.Reply(emb)
}

func checkinSummary(mood string, intensity int32, activity, note *string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Current mood: %s **%s** (%d/5)\n", moodEmoji[mood], strings.Title(mood), intensity)
	if activity != nil {
		fmt.Fprintf(&b, "Activity: %s\n", *activity)
	}
	if note != nil {
		fmt.Fprintf(&b, "Note: %s\n", *note)
	}
	return b.String()
}

func renderRecent(rows []gen.MoodCheckIn) string {
	if len(rows) == 0 {
		return "No check-ins yet."
	}
	var b strings.Builder
	for _, r := range rows {
		emoji := moodEmoji[r.Mood]
		if emoji == "" {
			emoji = "-"
		}
		fmt.Fprintf(&b, "%s **%s** (%d/5) <t:%d:R>\n", emoji, r.Mood, intensityOf(r), r.CreatedAt.Unix())
	}
	return b.String()
}

func intensityOf(r gen.MoodCheckIn) int {
	if r.Intensity == nil {
		return 3
	}
	return int(*r.Intensity)
}

func runInsights(ctx context.Context, c *Ctx) error {
	since := time.Now().AddDate(0, 0, -30)
	rows, err := c.Store.MoodCheckInsSince(ctx, c.UserID, since)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return c.Reply(infoEmbed("Mood insights", "No check-ins in the last 30 days. Try /checkin to get started."))
	}

	counts := map[string]int{}
	var sum, n int
	for _, r := range rows {
		counts[r.Mood]++
		if r.Intensity != nil {
			sum += int(*r.Intensity)
			n++
		}
	}
	type kv struct {
		k string
		v int
	}
	var top []kv
	for k, v := range counts {
		top = append(top, kv{k, v})
	}
	sort.Slice(top, func(i, j int) bool { return top[i].v > top[j].v })

	var b strings.Builder
	fmt.Fprintf(&b, "Check-ins in the last 30 days: **%d**\n", len(rows))
	if n > 0 {
		fmt.Fprintf(&b, "Average intensity: **%.1f/5**\n", float64(sum)/float64(n))
	}
	fmt.Fprintf(&b, "Current daily streak: **%d**\n\n", dailyStreak(rows))
	b.WriteString("Most common moods:\n")
	for i, t := range top {
		if i == 5 {
			break
		}
		emoji := moodEmoji[t.k]
		if emoji == "" {
			emoji = "-"
		}
		fmt.Fprintf(&b, "%s %s x%d\n", emoji, t.k, t.v)
	}
	return c.Reply(infoEmbed("Mood insights", b.String()))
}

func dailyStreak(rows []gen.MoodCheckIn) int {
	days := map[string]bool{}
	for _, r := range rows {
		days[r.CreatedAt.Format("2006-01-02")] = true
	}
	streak := 0
	day := time.Now()
	for {
		if days[day.Format("2006-01-02")] {
			streak++
			day = day.AddDate(0, 0, -1)
			continue
		}
		if streak == 0 && day.Format("2006-01-02") == time.Now().Format("2006-01-02") {
			day = day.AddDate(0, 0, -1)
			continue
		}
		break
	}
	return streak
}
