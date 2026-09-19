package db

import (
	"testing"
	"time"

	"github.com/CodeMeAPixel/Mellow/internal/db/gen"
)

func prefsWithQuiet(start, end int32, tz string) gen.UserPreferences {
	p := gen.UserPreferences{QuietStart: &start, QuietEnd: &end}
	if tz != "" {
		p.Timezone = &tz
	}
	return p
}

func TestInQuietHours(t *testing.T) {
	at := func(h int) time.Time { return time.Date(2026, 1, 15, h, 30, 0, 0, time.UTC) }

	overnight := prefsWithQuiet(22, 7, "")
	for h, want := range map[int]bool{21: false, 22: true, 23: true, 0: true, 6: true, 7: false, 12: false} {
		if got := InQuietHours(overnight, at(h)); got != want {
			t.Errorf("overnight window at %02d:30 = %v, want %v", h, got, want)
		}
	}

	daytime := prefsWithQuiet(13, 15, "")
	if !InQuietHours(daytime, at(13)) || InQuietHours(daytime, at(15)) {
		t.Error("same-day window boundaries wrong")
	}

	if InQuietHours(prefsWithQuiet(5, 5, ""), at(5)) {
		t.Error("equal start and end should mean off")
	}
	if InQuietHours(gen.UserPreferences{}, at(3)) {
		t.Error("unset window should never be quiet")
	}

	// 03:30 UTC is 22:30 the previous evening in New York (UTC-5 in January).
	if !InQuietHours(prefsWithQuiet(22, 7, "America/New_York"), at(3)) {
		t.Error("window should be evaluated in the user's timezone")
	}
	if InQuietHours(prefsWithQuiet(22, 7, "America/New_York"), at(15)) {
		t.Error("10:30 in New York is outside the window")
	}
}
