package changelog

import (
	_ "embed"
	"os"
	"regexp"
	"strings"
)

//go:embed PATCHNOTES.md
var embedded string

type Entry struct {
	Version string
	Title   string
	Content string
}

func Source() string {
	if p := os.Getenv("PATCHNOTES_PATH"); p != "" {
		if b, err := os.ReadFile(p); err == nil {
			return string(b)
		}
	}
	return embedded
}

var headingRE = regexp.MustCompile(`(?m)^##\s+(.+?)\s*$`)

func Parse(md string) []Entry {
	locs := headingRE.FindAllStringSubmatchIndex(md, -1)
	entries := make([]Entry, 0, len(locs))
	for i, loc := range locs {
		heading := md[loc[2]:loc[3]]
		bodyStart := loc[1]
		bodyEnd := len(md)
		if i+1 < len(locs) {
			bodyEnd = locs[i+1][0]
		}
		version, title := splitHeading(heading)
		e := Entry{
			Version: version,
			Title:   title,
			Content: strings.TrimSpace(md[bodyStart:bodyEnd]),
		}
		if e.Content == "" {
			continue
		}
		entries = append(entries, e)
	}
	return entries
}

func Latest(md string) (Entry, bool) {
	e := Parse(md)
	if len(e) == 0 {
		return Entry{}, false
	}
	return e[0], true
}

func splitHeading(h string) (version, title string) {
	h = strings.TrimSpace(h)
	var rest string
	for _, sep := range []string{" - ", " — ", " – ", ": ", " "} {
		if i := strings.Index(h, sep); i >= 0 {
			version, rest = h[:i], strings.TrimSpace(h[i+len(sep):])
			break
		}
	}
	if version == "" {
		version = h
	}
	version = strings.TrimPrefix(strings.TrimSpace(version), "v")
	version = strings.TrimPrefix(version, "V")
	if rest == "" {
		rest = "v" + version
	}
	return version, rest
}
