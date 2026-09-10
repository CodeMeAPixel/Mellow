package discord

import (
	"context"
	"errors"
	"time"

	"github.com/CodeMeAPixel/Mellow/internal/ai"
	"github.com/disgoorg/disgo/discord"
)

func memeCommands() []*Command {
	return []*Command{
		{
			Name: "memegen", Description: "Generate a wholesome meme.",
			Category: "Fun", Cooldown: 10 * time.Second,
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionString{
					Name: "template", Description: "Meme template",
					Choices: []discord.ApplicationCommandOptionChoiceString{
						{Name: "Drake", Value: "drake"},
						{Name: "Distracted", Value: "distracted"},
						{Name: "Expanding brain", Value: "expandingbrain"},
						{Name: "Two buttons", Value: "twobuttons"},
						{Name: "Change my mind", Value: "changemymind"},
						{Name: "This is fine", Value: "thisisfine"},
						{Name: "Spider-Man pointing", Value: "spiderman"},
					},
				},
				discord.ApplicationCommandOptionString{Name: "topic", Description: "What should it be about?"},
			},
			Run: func(ctx context.Context, c *Ctx) error {
				_ = c.Defer(false)
				m, err := c.AI.GenerateMeme(ctx, c.String("template"), c.String("topic"))
				if err != nil {
					if errors.Is(err, ai.ErrNoKey) || errors.Is(err, ai.ErrDisabled) {
						return c.ReplyText("Meme generation needs the AI to be configured.")
					}
					return err
				}
				title := m.Title
				if title == "" {
					title = "Here you go"
				}
				emb := infoEmbed(title, m.TopText+"\n\n"+m.BotText).WithImage(m.ImageURL)
				return c.Reply(emb)
			},
		},
	}
}
