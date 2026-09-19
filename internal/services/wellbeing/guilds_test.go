package wellbeing

import (
	"strings"
	"testing"
	"time"

	"github.com/CodeMeAPixel/Mellow/internal/db/gen"
)

func TestGuildPromptDue(t *testing.T) {
	i := func(v int32) *int32 { return &v }
	s := func(v string) *string { return &v }
	// Thursday 2026-01-15 10:30 UTC.
	now := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)

	daily := gen.Guild{PromptEnabled: true, PromptHour: 10}
	if !GuildPromptDue(daily, now) {
		t.Error("daily prompt should be due at its hour")
	}
	if GuildPromptDue(gen.Guild{PromptEnabled: false, PromptHour: 10}, now) {
		t.Error("disabled prompt must never be due")
	}
	if GuildPromptDue(gen.Guild{PromptEnabled: true, PromptHour: 11}, now) {
		t.Error("wrong hour")
	}

	weekly := gen.Guild{PromptEnabled: true, PromptHour: 10, PromptDay: i(int32(time.Thursday))}
	if !GuildPromptDue(weekly, now) {
		t.Error("weekly prompt should be due on its weekday")
	}
	weekly.PromptDay = i(int32(time.Friday))
	if GuildPromptDue(weekly, now) {
		t.Error("weekly prompt must wait for its weekday")
	}

	recent := now.Add(-2 * time.Hour)
	daily.LastPromptAt = &recent
	if GuildPromptDue(daily, now) {
		t.Error("should not repeat within the same window")
	}
	old := now.Add(-23 * time.Hour)
	daily.LastPromptAt = &old
	if !GuildPromptDue(daily, now) {
		t.Error("should post again the next day")
	}

	// 10:30 UTC is 05:30 in New York (UTC-5 in January).
	ny := gen.Guild{PromptEnabled: true, PromptHour: 5, PromptTimezone: s("America/New_York")}
	if !GuildPromptDue(ny, now) {
		t.Error("hour should be read in the server's timezone")
	}
	if GuildPromptDue(gen.Guild{PromptEnabled: true, PromptHour: 10, PromptTimezone: s("Not/AZone")}, now) != true {
		t.Error("an invalid timezone should fall back to UTC")
	}
}

func TestGuildPromptText(t *testing.T) {
	text := GuildPromptText(time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC))
	for _, want := range []string{"/checkin", "/guided", "Only you see"} {
		if !strings.Contains(text, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
}
