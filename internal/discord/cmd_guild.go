package discord

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/CodeMeAPixel/Mellow/internal/db"
	"github.com/CodeMeAPixel/Mellow/internal/db/gen"
	"github.com/disgoorg/disgo/discord"
)

func guildCommands() []*Command {
	channelOpt := func(name, desc string) discord.ApplicationCommandOptionChannel {
		return discord.ApplicationCommandOptionChannel{Name: name, Description: desc}
	}
	return []*Command{
		{
			Name: "guildsettings", Description: "Configure Mellow for this server.",
			Category: "Guild", GuildOnly: true, Cooldown: 5 * time.Second,
			RequiredPerms: []discord.Permissions{discord.PermissionManageGuild},
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionSubCommand{Name: "view", Description: "Show the current settings."},
				discord.ApplicationCommandOptionSubCommand{
					Name: "set", Description: "Change one or more settings.",
					Options: []discord.ApplicationCommandOption{
						channelOpt("checkin_channel", "Channel for check-in logs"),
						channelOpt("mod_alert_channel", "Channel for crisis and mod alerts"),
						channelOpt("mod_log_channel", "Channel for moderation logs"),
						channelOpt("system_channel", "Channel for Mellow system messages"),
						discord.ApplicationCommandOptionRole{Name: "moderator_role", Description: "Role treated as server moderators"},
						discord.ApplicationCommandOptionBool{Name: "check_ins", Description: "Enable check-in features here"},
						discord.ApplicationCommandOptionBool{Name: "ghost_letters", Description: "Enable ghost letters here"},
						discord.ApplicationCommandOptionBool{Name: "crisis_alerts", Description: "Enable crisis alerts here"},
						discord.ApplicationCommandOptionBool{Name: "system_logs", Description: "Enable system logs here"},
						discord.ApplicationCommandOptionBool{Name: "context_logging", Description: "Allow logging messages for AI context"},
						discord.ApplicationCommandOptionString{Name: "language", Description: "Preferred language code, e.g. en"},
					},
				},
			},
			Run: runGuildSettings,
		},
	}
}

func runGuildSettings(ctx context.Context, c *Ctx) error {
	if c.GuildID == nil {
		return c.ReplyEphemeral("This command can only be used in a server.")
	}
	if c.Sub() == "view" {
		g, err := c.Store.GetGuild(ctx, *c.GuildID)
		if err != nil {
			return c.Reply(infoEmbed("Server settings", "Mellow has no saved settings for this server yet."))
		}
		return c.Reply(infoEmbed("Server settings", renderGuild(g)))
	}

	var upd db.GuildSettingsUpdate
	changed := false
	if v, ok := c.Data.OptChannel("checkin_channel"); ok {
		s := v.ID.String()
		upd.CheckInChannelID = &s
		changed = true
	}
	if v, ok := c.Data.OptChannel("mod_alert_channel"); ok {
		s := v.ID.String()
		upd.ModAlertChannelID = &s
		changed = true
	}
	if v, ok := c.Data.OptChannel("mod_log_channel"); ok {
		s := v.ID.String()
		upd.ModLogChannelID = &s
		changed = true
	}
	if v, ok := c.Data.OptChannel("system_channel"); ok {
		s := v.ID.String()
		upd.SystemChannelID = &s
		changed = true
	}
	if v, ok := c.Data.OptRole("moderator_role"); ok {
		s := v.ID.String()
		upd.ModeratorRoleID = &s
		changed = true
	}
	if v, ok := c.Data.OptBool("check_ins"); ok {
		upd.EnableCheckIns = &v
		changed = true
	}
	if v, ok := c.Data.OptBool("ghost_letters"); ok {
		upd.EnableGhostLetters = &v
		changed = true
	}
	if v, ok := c.Data.OptBool("crisis_alerts"); ok {
		upd.EnableCrisisAlerts = &v
		changed = true
	}
	if v, ok := c.Data.OptBool("system_logs"); ok {
		upd.SystemLogsEnabled = &v
		changed = true
	}
	if v, ok := c.Data.OptBool("context_logging"); ok {
		inv := !v
		upd.DisableContextLogging = &inv
		changed = true
	}
	if v := c.String("language"); v != "" {
		upd.Language = &v
		changed = true
	}
	if !changed {
		return c.ReplyEphemeral("Nothing to change. Provide at least one option.")
	}

	if _, err := c.Store.GetGuild(ctx, *c.GuildID); err != nil {
		guild, ok := c.Event.Guild()
		name := "server"
		var owner int64
		if ok {
			name = guild.Name
			owner = int64(guild.OwnerID)
		}
		if _, err := c.Store.UpsertGuild(ctx, *c.GuildID, name, owner); err != nil {
			return err
		}
	}

	g, err := c.Store.UpdateGuildSettings(ctx, *c.GuildID, upd)
	if err != nil {
		return err
	}
	return c.Reply(successEmbed("Server settings updated", renderGuild(g)))
}

func renderGuild(g gen.Guild) string {
	ch := func(p *string) string {
		if p == nil || *p == "" {
			return "not set"
		}
		return "<#" + *p + ">"
	}
	role := func(p *string) string {
		if p == nil || *p == "" {
			return "not set"
		}
		return "<@&" + *p + ">"
	}
	onOff := func(b bool) string {
		if b {
			return "on"
		}
		return "off"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Check-in channel: %s\n", ch(g.CheckInChannelId))
	fmt.Fprintf(&b, "Mod alert channel: %s\n", ch(g.ModAlertChannelId))
	fmt.Fprintf(&b, "Mod log channel: %s\n", ch(g.ModLogChannelId))
	fmt.Fprintf(&b, "System channel: %s\n", ch(g.SystemChannelId))
	fmt.Fprintf(&b, "Moderator role: %s\n", role(g.ModeratorRoleId))
	fmt.Fprintf(&b, "Check-ins: %s | Ghost letters: %s | Crisis alerts: %s\n",
		onOff(g.EnableCheckIns), onOff(g.EnableGhostLetters), onOff(g.EnableCrisisAlerts))
	fmt.Fprintf(&b, "System logs: %s | Context logging: %s\n",
		onOff(g.SystemLogsEnabled), onOff(!g.DisableContextLogging))
	lang := "en"
	if g.Language != nil && *g.Language != "" {
		lang = *g.Language
	}
	fmt.Fprintf(&b, "Language: %s", lang)
	return b.String()
}
