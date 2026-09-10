package discord

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
)

func guildDiagCommands() []*Command {
	return []*Command{
		{
			Name: "guildcontext", Description: "Show how Mellow handles context and privacy in this server.",
			Category: "Guild", GuildOnly: true, Cooldown: 10 * time.Second,
			Run: runGuildContext,
		},
		{
			Name: "guilddebug", Description: "Show Mellow's stored configuration and status for this server.",
			Category: "Guild", GuildOnly: true, Cooldown: 10 * time.Second,
			RequiredPerms: []discord.Permissions{discord.PermissionManageGuild},
			Run:           runGuildDebug,
		},
	}
}

func runGuildContext(ctx context.Context, c *Ctx) error {
	if c.GuildID == nil {
		return c.ReplyEphemeral("This command can only be used in a server.")
	}
	g, err := c.Store.GetGuild(ctx, *c.GuildID)
	logging := "on"
	if err == nil && g.DisableContextLogging {
		logging = "off"
	}
	alerts := "on"
	if err == nil && !g.EnableCrisisAlerts {
		alerts = "off"
	}
	body := fmt.Sprintf(
		"Context logging in this server: **%s**\nCrisis alerts: **%s**\n\n"+
			"Mellow only reads messages that mention it or are sent in DMs. It never reads all messages. "+
			"Members can further limit logging with /preferences set context_logging:false.",
		logging, alerts,
	)
	return c.Reply(infoEmbed("Server context and privacy", body))
}

func runGuildDebug(ctx context.Context, c *Ctx) error {
	if c.GuildID == nil {
		return c.ReplyEphemeral("This command can only be used in a server.")
	}
	g, err := c.Store.GetGuild(ctx, *c.GuildID)
	if err != nil {
		return c.Reply(infoEmbed("Server debug", "No stored configuration for this server. Run /guildsettings set to create one."))
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Guild ID: `%d`\n", g.ID)
	fmt.Fprintf(&b, "Name: %s\n", g.Name)
	fmt.Fprintf(&b, "Owner ID: `%d`\n", g.OwnerId)
	fmt.Fprintf(&b, "Joined: <t:%d:R>\n", g.JoinedAt.Unix())
	fmt.Fprintf(&b, "Banned: %v\n\n", g.IsBanned)
	b.WriteString(renderGuild(g))
	return c.Reply(infoEmbed("Server debug", b.String()))
}
