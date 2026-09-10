package discord

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/CodeMeAPixel/Mellow/internal/ai"
	"github.com/disgoorg/disgo/discord"
)

func crisisCommand() *Command {
	return &Command{
		Name: "crisis", Description: "Crisis support tools and resources.",
		Category: "Crisis", Cooldown: 8 * time.Second,
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionSubCommand{
				Name: "analyze", Description: "Analyze a message for crisis indicators and get support.",
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionString{Name: "message", Description: "The message to analyze", Required: true},
				},
			},
			discord.ApplicationCommandOptionSubCommand{
				Name: "resources", Description: "Get crisis support resources and recommendations.",
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionString{Name: "situation", Description: "Briefly, what is happening?", Required: true},
				},
			},
			discord.ApplicationCommandOptionSubCommand{
				Name: "history", Description: "View your recent crisis events.",
			},
		},
		Run: runCrisis,
	}
}

func runCrisis(ctx context.Context, c *Ctx) error {
	switch c.Sub() {
	case "analyze":
		return runCrisisAnalyze(ctx, c)
	case "resources":
		return runCrisisResources(ctx, c)
	case "history":
		return runCrisisHistory(ctx, c)
	}
	return c.ReplyEphemeral("Unknown crisis subcommand.")
}

func runCrisisAnalyze(ctx context.Context, c *Ctx) error {
	msg := c.String("message")
	_ = c.Defer(true)

	res, err := c.AI.AnalyzeCrisis(ctx, msg)
	if err != nil {
		return err
	}

	if res.NeedsSupport {
		details := res.Summary
		_, _ = c.Store.CreateCrisisEvent(ctx, c.UserID, &details, res.Level == "critical")
		if res.Respond {
			c.Bot.syslog.Crisis(ctx, c.UserID, c.GuildID, res.Level, res.Summary)
		}
	}

	body := describeCrisis(res)
	if res.NeedsSupport {
		body += "\n\n" + ai.CrisisResourceBlock
	}
	emb := infoEmbed("Crisis analysis", body)
	switch res.Level {
	case "high", "critical":
		emb = emb.WithColor(colorError)
	case "medium":
		emb = emb.WithColor(colorWarning)
	}
	return c.Reply(emb)
}

func runCrisisResources(ctx context.Context, c *Ctx) error {
	situation := c.String("situation")
	_ = c.Defer(true)
	return c.Reply(infoEmbed("Crisis support resources", c.AI.CrisisResourcesText(ctx, situation)).WithColor(colorWarning))
}

func runCrisisHistory(ctx context.Context, c *Ctx) error {
	_ = c.Defer(true)
	rows, err := c.Store.CrisisEventsForUser(ctx, c.UserID, 10)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return c.Reply(infoEmbed("Crisis history", "No crisis events on record."))
	}
	var b strings.Builder
	for _, r := range rows {
		mark := ""
		if r.Escalated {
			mark = " (escalated)"
		}
		detail := ""
		if r.Details != nil {
			detail = " - " + *r.Details
		}
		fmt.Fprintf(&b, "<t:%d:R>%s%s\n", r.DetectedAt.Unix(), mark, detail)
	}
	return c.Reply(infoEmbed("Your recent crisis events", b.String()))
}
