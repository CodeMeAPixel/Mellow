package ai

import "strings"

const discordMessageLimit = 2000

func formatForDiscord(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	for strings.Contains(s, "\n\n\n") {
		s = strings.ReplaceAll(s, "\n\n\n", "\n\n")
	}
	if len(s) > discordMessageLimit {
		s = s[:discordMessageLimit-3] + "..."
	}
	return s
}
