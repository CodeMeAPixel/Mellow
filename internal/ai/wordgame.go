package ai

import (
	"context"
	"strings"
)

const wordGameSystem = "You are Mellow, running a gentle word game for mental wellness. Keep it encouraging and clear."

func (c *Client) GenerateWordGame(ctx context.Context, kind, difficulty string) (prompt, answer string, err error) {
	instruction := wordGameInstruction(kind, difficulty) +
		"\n\nEnd your response with a line 'CORRECT_ANSWER: ...' listing the accepted answers separated by commas. That line will be hidden from the player."
	out, err := c.oneShot(ctx, wordGameSystem, instruction)
	if err != nil {
		return "", "", err
	}
	if i := strings.LastIndex(out, "CORRECT_ANSWER:"); i >= 0 {
		answer = strings.TrimSpace(out[i+len("CORRECT_ANSWER:"):])
		out = strings.TrimSpace(out[:i])
	}
	return out, answer, nil
}

func (c *Client) EvaluateWordGame(ctx context.Context, kind, correct, guess string) (string, error) {
	user := "Game type: " + kind + "\nAccepted answers: " + correct + "\nPlayer's answer: " + guess +
		"\n\nEvaluate the answer. Start with 'Correct!' or 'Not quite, but'. Be supportive, give a hint if wrong, keep it short."
	return c.oneShot(ctx, wordGameSystem, user)
}

func wordGameInstruction(kind, difficulty string) string {
	if difficulty == "" {
		difficulty = "medium"
	}
	switch kind {
	case "rhyme":
		return "Give the player one word and ask them to find words that rhyme with it. Difficulty: " + difficulty + "."
	case "puzzle":
		return "Give the player a scrambled word to unscramble. Use a positive, encouraging word. Difficulty: " + difficulty + "."
	case "positive":
		return "Ask the player to name a word related to wellbeing, calm, or growth that fits a clue you give. Difficulty: " + difficulty + "."
	default:
		return "Give the player 4 words and ask them to find the connection between them. Include uplifting words. Difficulty: " + difficulty + "."
	}
}

func FuzzyMatch(correct, guess string) bool {
	g := strings.ToLower(strings.TrimSpace(guess))
	if g == "" {
		return false
	}
	for _, part := range strings.Split(strings.ToLower(correct), ",") {
		p := strings.TrimSpace(part)
		if p == "" {
			continue
		}
		if strings.Contains(g, p) || strings.Contains(p, g) {
			return true
		}
	}
	return false
}
