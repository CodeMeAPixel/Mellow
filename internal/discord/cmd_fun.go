package discord

import (
	"context"
	"errors"
	"time"

	"github.com/CodeMeAPixel/Mellow/internal/ai"
	"github.com/disgoorg/disgo/discord"
)

func funCommands() []*Command {
	fallbacks := map[string]string{
		"joke":       "Why did the scarecrow win an award? Because it was outstanding in its field.",
		"compliment": "You showed up today, and that takes real strength. That matters.",
		"trivia":     "Sea otters hold hands while they sleep so they do not drift apart.",
		"wyr":        "Would you rather always have a calm morning or a peaceful evening?",
	}
	mk := func(name, kind, desc string) *Command {
		return &Command{
			Name: name, Description: desc, Category: "Fun", Cooldown: 5 * time.Second, PremiumCooldown: 2 * time.Second,
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionString{Name: "topic", Description: "Optional topic"},
			},
			Run: func(ctx context.Context, c *Ctx) error {
				_ = c.Defer(false)
				out, err := c.AI.Fun(ctx, kind, c.String("topic"))
				if err != nil {
					if errors.Is(err, ai.ErrNoKey) || errors.Is(err, ai.ErrDisabled) {
						return c.Reply(infoEmbed(desc, fallbacks[kind]))
					}
					return err
				}
				return c.Reply(infoEmbed(desc, out))
			},
		}
	}
	return []*Command{
		mk("joke", "joke", "Get a wholesome joke."),
		mk("compliment", "compliment", "Receive a genuine compliment."),
		mk("trivia", "trivia", "Get a positive trivia fact."),
		mk("wouldyourather", "wyr", "Get a light would-you-rather question."),
	}
}
