package ai

import (
	"context"
	"testing"
)

func offlineClient() *Client { return &Client{live: false} }

func TestScreenBenign(t *testing.T) {
	for _, s := range []string{
		"hey mellow how are you",
		"i had a rough day at work",
		"can you help me with a breathing exercise",
		"that meeting was so boring i could die",
	} {
		if sc := screenCrisisKeywords(s); sc.hasKeywords || sc.hasPatterns {
			t.Errorf("benign %q flagged: %+v", s, sc)
		}
	}
}

func TestAnalyzeOfflineTiers(t *testing.T) {
	c := offlineClient()

	res, _ := c.AnalyzeCrisis(context.Background(), "the traffic today was suicide honestly")
	if !res.NeedsSupport {
		t.Fatalf("keyword should log: %+v", res)
	}
	if res.Respond {
		t.Errorf("lone keyword must NOT trigger a crisis reply: %+v", res)
	}
	if res.Level != "medium" {
		t.Errorf("lone keyword level = %q, want medium", res.Level)
	}

	res, _ = c.AnalyzeCrisis(context.Background(), "i want to kill myself tonight, i can't do this")
	if !res.Respond || res.Level != "high" {
		t.Fatalf("clear intent+plan must trigger reply at high: %+v", res)
	}
}

func TestCrisisResponseTemplates(t *testing.T) {
	c := offlineClient()

	high := c.CrisisResponse(context.Background(), "high", "i want to end my life")
	if !contains(high, "988") || !contains(high, "findahelpline.com") {
		t.Errorf("high template missing verified resources: %q", high)
	}
	if high != criticalCrisisMessage {
		t.Errorf("high must be the fixed message verbatim")
	}

	med := c.CrisisResponse(context.Background(), "medium", "i feel hopeless")
	if !contains(med, "988") {
		t.Errorf("medium fallback missing resources: %q", med)
	}
}

func TestMethodContentFilter(t *testing.T) {
	positives := []string{
		"you could take 30 pills",
		"here is how to tie a noose",
		"the lethal dose is",
		"step by step you would",
	}
	for _, s := range positives {
		if !containsMethodContent(s) {
			t.Errorf("should be filtered: %q", s)
		}
	}
	negatives := []string{
		"i'm here for you",
		"please call 988",
		"you deserve support",
	}
	for _, s := range negatives {
		if containsMethodContent(s) {
			t.Errorf("false positive: %q", s)
		}
	}
}

func TestResourcesTextOfflineIsFixed(t *testing.T) {
	c := offlineClient()
	out := c.CrisisResourcesText(context.Background(), "everything feels pointless")
	if !contains(out, "988") || !contains(out, CrisisResourceBlock) {
		t.Errorf("resources text must contain the fixed block: %q", out)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (haystack == needle || indexOf(haystack, needle) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
