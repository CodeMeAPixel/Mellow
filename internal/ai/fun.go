package ai

import "context"

var funPrompts = map[string]string{
	"joke":       "Tell one short, wholesome, genuinely funny joke. Keep it clean and kind. No preamble.",
	"compliment": "Give the user one sincere, specific, uplifting compliment. Keep it to one or two sentences.",
	"trivia":     "Share one interesting, positive trivia fact and a one line explanation. No preamble.",
	"wyr":        "Pose one light, fun 'would you rather' question with two balanced options. Keep it wholesome. No preamble.",
}

func (c *Client) Fun(ctx context.Context, kind, topic string) (string, error) {
	instruction, ok := funPrompts[kind]
	if !ok {
		instruction = "Share something light and positive."
	}
	if topic != "" {
		instruction += " Topic: " + topic
	}
	system := "You are Mellow, a warm mental health companion with a gentle sense of humour. Keep replies short and kind."
	return c.oneShot(ctx, system, instruction)
}
