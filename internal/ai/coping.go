package ai

import (
	"context"
	"fmt"
)

var copingToolPrompts = map[string]string{
	"breathing":    "Guide the user through a short breathing exercise (for example 4-7-8 or box breathing). Give clear step by step instructions they can follow right now.",
	"grounding":    "Guide the user through a 5-4-3-2-1 grounding exercise, naming each sense in turn. Keep it calm and paced.",
	"affirmations": "Offer the user two or three genuine, personalised affirmations. Avoid cliches. Keep them warm and believable.",
	"challenge":    "Give the user one small, achievable self-care or coping challenge for today, with a sentence on why it helps.",
	"distraction":  "Offer the user a light, healthy distraction: a short game idea, a fun fact, or a small creative prompt.",
	"music":        "Suggest a few calming music styles, artists, or playlists suited to relaxation. Explain briefly why each fits.",
	"gratitude":    "Help the user notice something to be grateful for right now with a gentle prompt or two.",
	"plan":         "Help the user shape a short personalised coping plan: a few strategies they can turn to when things get hard.",
}

func (c *Client) Coping(ctx context.Context, tool, feeling string) (string, error) {
	cfg := c.config(ctx)
	if !cfg.CopingTools {
		return "", fmt.Errorf("coping tools are disabled")
	}
	instruction, ok := copingToolPrompts[tool]
	if !ok {
		instruction = "Offer the user a supportive coping strategy suited to how they are feeling."
	}
	user := instruction
	if feeling != "" {
		user += "\n\nThe user says they are feeling: " + feeling
	}
	system := fallbackSystemPrompt
	if p := cfg.Prompt; p != "" {
		system = p
	}
	system += personalityInstructions("gentle")
	return c.oneShot(ctx, system, user)
}
