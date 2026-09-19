package discord

import (
	"strings"
	"testing"
	"time"
)

func TestGuidedExercisesAreWellFormed(t *testing.T) {
	for _, e := range guidedExercises {
		steps := guidedSteps(e.value)
		if len(steps) < 3 {
			t.Fatalf("%s has too few steps", e.value)
		}
		if guidedTitle(e.value) == "" {
			t.Errorf("%s has no title", e.value)
		}

		var total time.Duration
		for i, s := range steps {
			if strings.TrimSpace(s.text) == "" {
				t.Errorf("%s step %d is empty", e.value, i)
			}
			if i < len(steps)-1 && s.hold <= 0 {
				t.Errorf("%s step %d needs a hold before the next step", e.value, i)
			}
			total += s.hold
		}
		if steps[len(steps)-1].hold != 0 {
			t.Errorf("%s last step should not wait", e.value)
		}
		if !strings.Contains(steps[len(steps)-1].text, "/crisis resources") {
			t.Errorf("%s should end by pointing to crisis resources", e.value)
		}
		// Interaction tokens last 15 minutes and edits must finish well inside that.
		if total > 8*time.Minute {
			t.Errorf("%s runs %s, too long for one interaction token", e.value, total)
		}
	}
	if guidedSteps("nope") != nil {
		t.Error("unknown exercise should return nil")
	}
}
