package discord

import (
	"context"
	"fmt"
	"time"

	"github.com/disgoorg/disgo/discord"
)

func contextCommands() []*Command {
	return []*Command{
		{
			Name: "context", Description: "See or clear what Mellow remembers about your conversations.",
			Category: "Users", Cooldown: 10 * time.Second,
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionSubCommand{Name: "view", Description: "See how much context is stored and your privacy settings."},
				discord.ApplicationCommandOptionSubCommand{Name: "clear", Description: "Delete all stored conversation history for you."},
			},
			Run: runContext,
		},
	}
}

func runContext(ctx context.Context, c *Ctx) error {
	prefs, err := c.Store.EnsureUserPreferences(ctx, c.UserID)
	if err != nil {
		return err
	}

	if c.Sub() == "clear" {
		n, err := c.Store.DeleteConversationForUser(ctx, c.UserID)
		if err != nil {
			return err
		}
		return c.ReplyEphemeral(fmt.Sprintf("Deleted %d stored messages. Mellow no longer has that conversation history.", n))
	}

	recent, _ := c.Store.RecentConversationForUser(ctx, c.UserID, nil, 1000)
	logging := "on"
	if prefs.DisableContextLogging {
		logging = "off"
	}
	body := fmt.Sprintf(
		"Stored messages: **%d**\nContext logging: **%s**\nCrisis detection: **%s**\n\nUse /context clear to erase this, or /preferences set context_logging:false to stop new logging.",
		len(recent), logging, onOffBool(!prefs.DisableCrisisDetection),
	)
	return c.Reply(infoEmbed("Your conversation context", body))
}

func onOffBool(b bool) string {
	if b {
		return "on"
	}
	return "off"
}
