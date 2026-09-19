package discord

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/CodeMeAPixel/Mellow/internal/db/gen"
)

var hashtag = regexp.MustCompile(`#([\p{L}\p{N}_]{1,32})`)

func journalTags(content string) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range hashtag.FindAllStringSubmatch(content, -1) {
		t := strings.ToLower(m[1])
		if !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	return out
}

func filterJournal(entries []gen.JournalEntry, search, tag string) []gen.JournalEntry {
	search = strings.ToLower(strings.TrimSpace(search))
	tag = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(tag), "#"))

	var out []gen.JournalEntry
	for _, e := range entries {
		if search != "" && !strings.Contains(strings.ToLower(e.Content), search) {
			continue
		}
		if tag != "" {
			found := false
			for _, t := range journalTags(e.Content) {
				if t == tag {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		out = append(out, e)
	}
	return out
}

func renderJournalMatches(matches []gen.JournalEntry, limit int) string {
	var b strings.Builder
	for i, r := range matches {
		if i == limit {
			fmt.Fprintf(&b, "...and %d more. Narrow your search to see them.", len(matches)-limit)
			break
		}
		body := []rune(r.Content)
		text := string(body)
		if len(body) > 200 {
			text = string(body[:200]) + "..."
		}
		fmt.Fprintf(&b, "<t:%d:D>\n%s\n\n", r.CreatedAt.Unix(), text)
	}
	return strings.TrimSpace(b.String())
}
