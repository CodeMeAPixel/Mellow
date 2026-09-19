package wellbeing

import (
	"strings"
	"testing"
	"time"

	"github.com/CodeMeAPixel/Mellow/internal/db/gen"
)

func TestBuildRecap(t *testing.T) {
	i := func(v int32) *int32 { return &v }
	rows := []gen.MoodCheckIn{
		{Mood: "calm", Intensity: i(4)},
		{Mood: "calm", Intensity: i(6)},
		{Mood: "sad", Intensity: i(8)},
		{Mood: "happy"},
	}
	out := BuildRecap(rows, "https://example.test/dashboard")

	for _, want := range []string{"4 times", "calm (2)", "Average intensity: 6.0", "https://example.test/dashboard", "weekly_recap:false"} {
		if !strings.Contains(out, want) {
			t.Errorf("recap missing %q:\n%s", want, out)
		}
	}
	if !strings.Contains(BuildRecap(rows[:1], ""), "once") {
		t.Error("single check-in should read naturally")
	}
}

func TestPromptForCyclesAndOptsOut(t *testing.T) {
	a := PromptFor(time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC))
	b := PromptFor(time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC))
	if a == b {
		t.Error("consecutive days should not repeat the same prompt")
	}
	if !strings.Contains(a, "daily_prompt:false") {
		t.Error("prompt should say how to turn it off")
	}
}
