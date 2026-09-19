package discord

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/CodeMeAPixel/Mellow/internal/ai"
	"github.com/CodeMeAPixel/Mellow/internal/db"
	"github.com/CodeMeAPixel/Mellow/internal/db/gen"
	"github.com/CodeMeAPixel/Mellow/internal/helplines"
	"github.com/disgoorg/disgo/discord"
)

func countryChoices() []discord.ApplicationCommandOptionChoiceString {
	cs := helplines.Countries()
	out := make([]discord.ApplicationCommandOptionChoiceString, 0, len(cs)+1)
	for _, c := range cs {
		out = append(out, discord.ApplicationCommandOptionChoiceString{Name: c.Name, Value: c.Code})
	}
	return append(out, discord.ApplicationCommandOptionChoiceString{Name: "Not listed / clear", Value: "none"})
}

var personalityChoices = []discord.ApplicationCommandOptionChoiceString{
	{Name: "Gentle", Value: "gentle"},
	{Name: "Supportive", Value: "supportive"},
	{Name: "Direct", Value: "direct"},
	{Name: "Playful", Value: "playful"},
	{Name: "Professional", Value: "professional"},
	{Name: "Encouraging", Value: "encouraging"},
	{Name: "Coach (Plus)", Value: "coach"},
	{Name: "Reflective (Plus)", Value: "reflective"},
	{Name: "Minimal (Plus)", Value: "minimal"},
}

func userCommands() []*Command {
	return []*Command{
		{
			Name: "preferences", Description: "View or change your Mellow preferences.",
			Category: "Users", Cooldown: 5 * time.Second,
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionSubCommand{Name: "view", Description: "View your current preferences."},
				discord.ApplicationCommandOptionSubCommand{
					Name: "set", Description: "Change one or more preferences.",
					Options: []discord.ApplicationCommandOption{
						discord.ApplicationCommandOptionString{Name: "personality", Description: "AI personality", Choices: personalityChoices},
						discord.ApplicationCommandOptionString{Name: "timezone", Description: "IANA timezone, e.g. Europe/London"},
						discord.ApplicationCommandOptionString{Name: "country", Description: "Your country, so crisis resources match where you are", Choices: countryChoices()},
						discord.ApplicationCommandOptionInt{Name: "checkin_interval", Description: "Minutes between check-in reminders"},
						discord.ApplicationCommandOptionBool{Name: "reminders", Description: "Enable check-in reminders"},
						discord.ApplicationCommandOptionInt{Name: "quiet_start", Description: "Hour (0-23, your time) when Mellow stops messaging you"},
						discord.ApplicationCommandOptionInt{Name: "quiet_end", Description: "Hour (0-23, your time) when Mellow may message you again"},
						discord.ApplicationCommandOptionBool{Name: "quiet_off", Description: "Turn quiet hours off"},
						discord.ApplicationCommandOptionBool{Name: "weekly_recap", Description: "Get a gentle weekly recap of your check-ins by DM"},
						discord.ApplicationCommandOptionBool{Name: "daily_prompt", Description: "Get one reflection prompt by DM each day"},
						discord.ApplicationCommandOptionString{Name: "persona", Description: "Plus: describe the conversation style you want (or type clear)"},
						discord.ApplicationCommandOptionBool{Name: "context_logging", Description: "Allow logging messages for AI context"},
						discord.ApplicationCommandOptionBool{Name: "crisis_detection", Description: "Enable crisis detection on your messages"},
						discord.ApplicationCommandOptionBool{Name: "crisis_dms", Description: "Allow Mellow to DM you crisis support"},
					},
				},
			},
			Run: runPreferences,
		},
		{
			Name: "timemode", Description: "Set your timezone so Mellow can adjust its tone.",
			Category: "Users", Cooldown: 5 * time.Second,
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionString{Name: "timezone", Description: "IANA timezone, e.g. America/New_York", Required: true},
			},
			Run: runTimemode,
		},
		{
			Name: "profile", Description: "View your Mellow profile and activity.",
			Category: "Users", Cooldown: 8 * time.Second,
			Run: runProfile,
		},
		{
			Name: "feedback", Description: "Send feedback about Mellow.",
			Category: "Users", Cooldown: 30 * time.Second,
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionString{Name: "message", Description: "Your feedback", Required: true},
			},
			Run: func(ctx context.Context, c *Ctx) error {
				if err := c.Store.CreateFeedback(ctx, c.UserID, c.String("message")); err != nil {
					return err
				}
				return c.ReplyEphemeral("Thank you, your feedback has been recorded.")
			},
		},
		{
			Name: "report", Description: "Report a problem or a user to the Mellow team.",
			Category: "Users", Cooldown: 30 * time.Second,
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionString{Name: "message", Description: "What happened?", Required: true},
			},
			Run: func(ctx context.Context, c *Ctx) error {
				if err := c.Store.CreateReport(ctx, c.UserID, c.String("message")); err != nil {
					return err
				}
				return c.ReplyEphemeral("Your report has been submitted. Thank you for letting us know.")
			},
		},
		{
			Name: "ghostletter", Description: "Write an unsent letter to get feelings out, or read your past ones.",
			Category: "Coping", Cooldown: 8 * time.Second, PremiumCooldown: 4 * time.Second,
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionString{Name: "write", Description: "The letter text"},
			},
			Run: runGhostLetter,
		},
	}
}

func runPreferences(ctx context.Context, c *Ctx) error {
	if c.Sub() == "view" {
		p, err := c.Store.EnsureUserPreferences(ctx, c.UserID)
		if err != nil {
			return err
		}
		return c.Reply(infoEmbed("Your preferences", renderPrefs(p)))
	}

	var upd db.PrefsUpdate
	changed := false
	if v := c.String("personality"); v != "" {
		if ai.IsPlusPersonality(v) && !c.HasPlus(ctx) {
			return c.ReplyEphemeral(plusOnly("The " + v + " style"))
		}
		upd.AIPersonality = &v
		changed = true
	}
	if v := c.String("persona"); v != "" {
		if !c.HasPlus(ctx) {
			return c.ReplyEphemeral(plusOnly("A custom conversation style"))
		}
		if strings.EqualFold(strings.TrimSpace(v), "clear") {
			v = ""
		} else {
			v = ai.SanitizePersona(v)
		}
		upd.CustomPersona = &v
		changed = true
	}
	if v, ok := c.Data.OptBool("weekly_recap"); ok {
		upd.WeeklyRecap = &v
		changed = true
	}
	if v, ok := c.Data.OptBool("daily_prompt"); ok {
		upd.DailyPrompt = &v
		changed = true
	}
	quietStart, hasStart := c.Int("quiet_start")
	quietEnd, hasEnd := c.Int("quiet_end")
	quietOff, _ := c.Data.OptBool("quiet_off")
	if quietOff || hasStart || hasEnd {
		if !quietOff {
			if !hasStart || !hasEnd || quietStart < 0 || quietStart > 23 || quietEnd < 0 || quietEnd > 23 {
				return c.ReplyEphemeral("Set both `quiet_start` and `quiet_end` to hours between 0 and 23, for example 22 and 7.")
			}
			s, e := int32(quietStart), int32(quietEnd)
			if err := c.Store.SetQuietHours(ctx, c.UserID, &s, &e); err != nil {
				return err
			}
		} else if err := c.Store.SetQuietHours(ctx, c.UserID, nil, nil); err != nil {
			return err
		}
		changed = true
	}
	if v := c.String("timezone"); v != "" {
		if _, err := time.LoadLocation(v); err != nil {
			return c.ReplyEphemeral("That does not look like a valid IANA timezone.")
		}
		upd.Timezone = &v
		changed = true
	}
	if v := c.String("country"); v != "" {
		if v == "none" {
			v = ""
		}
		upd.Country = &v
		changed = true
	}
	if v, ok := c.Int("checkin_interval"); ok {
		if v < 30 || v > 10080 {
			return c.ReplyEphemeral("Check-in interval must be between 30 and 10080 minutes.")
		}
		n := int32(v)
		upd.CheckInInterval = &n
		changed = true
	}
	if v, ok := c.Data.OptBool("reminders"); ok {
		upd.RemindersEnabled = &v
		changed = true
	}
	if v, ok := c.Data.OptBool("context_logging"); ok {
		inv := !v
		upd.DisableContextLogging = &inv
		changed = true
	}
	if v, ok := c.Data.OptBool("crisis_detection"); ok {
		inv := !v
		upd.DisableCrisisDetection = &inv
		changed = true
	}
	if v, ok := c.Data.OptBool("crisis_dms"); ok {
		inv := !v
		upd.DisableCrisisSupportDMs = &inv
		changed = true
	}
	if !changed {
		return c.ReplyEphemeral("Nothing to change. Provide at least one option.")
	}

	p, err := c.Store.UpdateUserPreferences(ctx, c.UserID, upd)
	if err != nil {
		return err
	}
	return c.Reply(successEmbed("Preferences updated", renderPrefs(p)))
}

func runTimemode(ctx context.Context, c *Ctx) error {
	tz := c.String("timezone")
	if _, err := time.LoadLocation(tz); err != nil {
		return c.ReplyEphemeral("That does not look like a valid IANA timezone. Example: Europe/London")
	}
	if _, err := c.Store.UpdateUserPreferences(ctx, c.UserID, db.PrefsUpdate{Timezone: &tz}); err != nil {
		return err
	}
	return c.Reply(successEmbed("Timezone set", "Mellow will use "+tz+" when adjusting its tone."))
}

func runProfile(ctx context.Context, c *Ctx) error {
	counts, err := c.Store.ProfileCounts(ctx, c.UserID)
	if err != nil {
		return err
	}
	prefs, _ := c.Store.EnsureUserPreferences(ctx, c.UserID)

	var b strings.Builder
	if c.HasPlus(ctx) {
		fmt.Fprintf(&b, "%s **Mellow+ supporter**\n\n", mellowEmoji("mellow"))
	}
	fmt.Fprintf(&b, "Check-ins: **%d**\n", counts.CheckIns)
	fmt.Fprintf(&b, "Journal entries: **%d**\n", counts.Journal)
	fmt.Fprintf(&b, "Gratitude notes: **%d**\n", counts.Gratitude)
	fmt.Fprintf(&b, "Ghost letters: **%d**\n", counts.GhostLetters)
	if counts.CrisisEvents > 0 {
		fmt.Fprintf(&b, "Crisis events: **%d** (%d escalated)\n", counts.CrisisEvents, counts.Escalated)
	}
	b.WriteString("\n")
	b.WriteString(renderPrefs(prefs))

	last, err := c.Store.LastMoodCheckIn(ctx, c.UserID)
	if err == nil {
		emoji := moodEmoji[last.Mood]
		if emoji == "" {
			emoji = "-"
		}
		fmt.Fprintf(&b, "\nLatest mood: %s %s (%d/5) <t:%d:R>", emoji, last.Mood, intensityOf(last), last.CreatedAt.Unix())
	}
	return c.Reply(infoEmbed("Your Mellow profile", b.String()))
}

func runGhostLetter(ctx context.Context, c *Ctx) error {
	if text := c.String("write"); text != "" {
		if _, err := c.Store.CreateGhostLetter(ctx, c.UserID, text); err != nil {
			return err
		}
		_ = c.Store.RecordCopingToolUsage(ctx, c.UserID, "ghostletter")
		return c.ReplyEphemeral("Your letter has been sealed away. Sometimes writing it down is enough.")
	}
	rows, err := c.Store.GhostLettersForUser(ctx, c.UserID, 5)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return c.ReplyEphemeral("You have not written any ghost letters yet. Use /ghostletter write:...")
	}
	var b strings.Builder
	for _, r := range rows {
		body := r.Content
		if len(body) > 300 {
			body = body[:300] + "..."
		}
		fmt.Fprintf(&b, "<t:%d:D>\n%s\n\n", r.CreatedAt.Unix(), body)
	}
	return c.ReplyEphemeral(b.String())
}

func renderPrefs(p gen.UserPreferences) string {
	deref := func(s *string, dflt string) string {
		if s == nil || *s == "" {
			return dflt
		}
		return *s
	}
	onOff := func(b bool) string {
		if b {
			return "on"
		}
		return "off"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "AI personality: **%s**\n", deref(p.AiPersonality, "gentle"))
	fmt.Fprintf(&b, "Timezone: **%s**\n", deref(p.Timezone, "not set"))
	country := "not set"
	if c, ok := helplines.Lookup(deref(p.Country, "")); ok {
		country = c.Name
	}
	fmt.Fprintf(&b, "Country (for crisis resources): **%s**\n", country)
	fmt.Fprintf(&b, "Check-in interval: **%d min**\n", p.CheckInInterval)
	quiet := "off"
	if p.QuietStart != nil && p.QuietEnd != nil {
		quiet = fmt.Sprintf("%02d:00 to %02d:00", *p.QuietStart, *p.QuietEnd)
	}
	fmt.Fprintf(&b, "Quiet hours: **%s**\n", quiet)
	fmt.Fprintf(&b, "Weekly recap: **%s**\n", onOff(p.WeeklyRecap))
	fmt.Fprintf(&b, "Daily prompt: **%s**\n", onOff(p.DailyPrompt))
	if p.CustomPersona != nil && *p.CustomPersona != "" {
		fmt.Fprintf(&b, "Custom style: **%s**\n", *p.CustomPersona)
	}
	fmt.Fprintf(&b, "Reminders: **%s** (%s)\n", onOff(p.RemindersEnabled), deref(p.ReminderMethod, "dm"))
	fmt.Fprintf(&b, "Context logging: **%s**\n", onOff(!p.DisableContextLogging))
	fmt.Fprintf(&b, "Crisis detection: **%s**\n", onOff(!p.DisableCrisisDetection))
	fmt.Fprintf(&b, "Crisis support DMs: **%s**\n", onOff(!p.DisableCrisisSupportDMs))
	return b.String()
}
