package helplines

import (
	"strings"
	"testing"
)

func TestBlockFallsBackToDefault(t *testing.T) {
	for _, code := range []string{"", "ZZ", "  "} {
		if got := Block(code); got != DefaultBlock {
			t.Errorf("Block(%q) should be the default block", code)
		}
	}
}

func TestEveryCountryHasUsableResources(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range Countries() {
		if seen[c.Code] {
			t.Errorf("duplicate country code %s", c.Code)
		}
		seen[c.Code] = true

		if len(c.Code) != 2 || c.Name == "" {
			t.Errorf("bad country entry %+v", c)
		}
		b := Block(c.Code)
		if !strings.Contains(b, c.Emergency) {
			t.Errorf("%s block is missing the emergency number", c.Code)
		}
		if len(c.Lines) == 0 {
			t.Errorf("%s has no helplines", c.Code)
		}
		for _, l := range c.Lines {
			if !strings.Contains(b, l) {
				t.Errorf("%s block is missing helpline %q", c.Code, l)
			}
		}
		if !strings.Contains(b, "findahelpline.com") {
			t.Errorf("%s block should include the international directory", c.Code)
		}
	}
}

func TestLookupIsCaseInsensitive(t *testing.T) {
	if _, ok := Lookup(" gb "); !ok {
		t.Error("expected gb to resolve")
	}
}
