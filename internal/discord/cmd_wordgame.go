package discord

import (
	"context"
	"time"

	"github.com/disgoorg/disgo/discord"
)

func wordgameCommands() []*Command {
	return []*Command{
		{
			Name: "wordgame", Description: "Play a short word game for a mental break.",
			Category: "Fun", Cooldown: 8 * time.Second,
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionString{
					Name: "type", Description: "Game type",
					Choices: []discord.ApplicationCommandOptionChoiceString{
						{Name: "Association", Value: "association"},
						{Name: "Rhyme", Value: "rhyme"},
						{Name: "Puzzle", Value: "puzzle"},
						{Name: "Positive", Value: "positive"},
					},
				},
				discord.ApplicationCommandOptionString{
					Name: "difficulty", Description: "Difficulty",
					Choices: []discord.ApplicationCommandOptionChoiceString{
						{Name: "Easy", Value: "easy"},
						{Name: "Medium", Value: "medium"},
						{Name: "Hard", Value: "hard"},
					},
				},
			},
			Run: func(ctx context.Context, c *Ctx) error {
				kind := c.String("type")
				if kind == "" {
					kind = "association"
				}
				difficulty := c.String("difficulty")
				if difficulty == "" {
					difficulty = "medium"
				}
				_ = c.Defer(false)
				c.Bot.startWordGame(ctx, func(m discord.MessageCreate) {
					_, _ = c.Event.Client().Rest.CreateFollowupMessage(c.Event.ApplicationID(), c.Event.Token(), m)
				}, c.UserID, kind, difficulty)
				return nil
			},
		},
	}
}
