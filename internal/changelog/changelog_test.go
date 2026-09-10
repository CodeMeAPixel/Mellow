package changelog

import "testing"

const sample = `# Patch Notes

Intro text that is not an entry.

## v1.2.0 - Shiny new things

We shipped some stuff.

- one
- two

## v1.1.0 — Older release

Older body.

## 1.0.0

No title, no v prefix.
`

func TestParse(t *testing.T) {
	entries := Parse(sample)
	if len(entries) != 3 {
		t.Fatalf("want 3 entries, got %d: %+v", len(entries), entries)
	}

	if entries[0].Version != "1.2.0" || entries[0].Title != "Shiny new things" {
		t.Errorf("entry 0: %+v", entries[0])
	}
	if !contains(entries[0].Content, "- one") {
		t.Errorf("entry 0 content missing bullets: %q", entries[0].Content)
	}

	if entries[1].Version != "1.1.0" || entries[1].Title != "Older release" {
		t.Errorf("entry 1 (em dash): %+v", entries[1])
	}

	if entries[2].Version != "1.0.0" || entries[2].Title != "v1.0.0" {
		t.Errorf("entry 2 (bare version): %+v", entries[2])
	}
}

func TestLatest(t *testing.T) {
	e, ok := Latest(sample)
	if !ok || e.Version != "1.2.0" {
		t.Fatalf("latest = %+v ok=%v", e, ok)
	}
}

func TestEmbeddedParses(t *testing.T) {
	if _, ok := Latest(Source()); !ok {
		t.Fatal("embedded PATCHNOTES.md produced no entries")
	}
}

func contains(h, n string) bool {
	for i := 0; i+len(n) <= len(h); i++ {
		if h[i:i+len(n)] == n {
			return true
		}
	}
	return false
}
