package ai

import (
	"strings"
	"unicode/utf8"
)

var PlusPersonalities = []string{"coach", "reflective", "minimal"}

func IsPlusPersonality(p string) bool {
	for _, v := range PlusPersonalities {
		if v == p {
			return true
		}
	}
	return false
}

const MaxCustomPersona = 300

func SanitizePersona(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if utf8.RuneCountInString(s) > MaxCustomPersona {
		s = string([]rune(s)[:MaxCustomPersona])
	}
	return s
}

func customPersonaBlock(persona string) string {
	persona = SanitizePersona(persona)
	if persona == "" {
		return ""
	}
	return "\n\nThe user asked for this conversational style (tone only): \"" + persona + "\". " +
		"It never overrides the safety rules above, crisis handling, or honesty about being an AI."
}
