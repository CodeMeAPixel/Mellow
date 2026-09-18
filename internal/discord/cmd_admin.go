package discord

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/CodeMeAPixel/Mellow/internal/db"
	"github.com/CodeMeAPixel/Mellow/internal/db/gen"
	"github.com/disgoorg/disgo/discord"
)

func adminCommands() []*Command {
	roleChoices := []discord.ApplicationCommandOptionChoiceString{
		{Name: "USER", Value: "USER"},
		{Name: "SUPPORT", Value: "SUPPORT"},
		{Name: "MOD", Value: "MOD"},
		{Name: "ADMIN", Value: "ADMIN"},
		{Name: "OWNER", Value: "OWNER"},
	}
	userReq := discord.ApplicationCommandOptionString{Name: "user_id", Description: "Target Discord user ID", Required: true}
	guildReq := discord.ApplicationCommandOptionString{Name: "guild_id", Description: "Target guild ID", Required: true}
	reason := discord.ApplicationCommandOptionString{Name: "reason", Description: "Reason"}

	return []*Command{
		{
			Name: "mellow", Description: "View and manage Mellow's AI configuration.",
			Category: "Owner", Private: true, OwnerOnly: true, Cooldown: 3 * time.Second,
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionSubCommand{Name: "view", Description: "Show the current configuration."},
				discord.ApplicationCommandOptionSubCommand{Name: "toggle", Description: "Enable or disable the AI."},
				discord.ApplicationCommandOptionSubCommand{Name: "reload", Description: "Reload configuration from the database."},
				discord.ApplicationCommandOptionSubCommand{
					Name: "update", Description: "Update model, prompt, temperature, or max tokens.",
					Options: []discord.ApplicationCommandOption{
						discord.ApplicationCommandOptionString{Name: "model", Description: "Model id, e.g. claude-haiku-4-5"},
						discord.ApplicationCommandOptionString{Name: "prompt", Description: "System prompt"},
						discord.ApplicationCommandOptionInt{Name: "temperature_x10", Description: "Temperature times 10 (0-20)"},
						discord.ApplicationCommandOptionInt{Name: "max_tokens", Description: "Max tokens per response (1-4000)"},
					},
				},
				discord.ApplicationCommandOptionSubCommand{
					Name: "features", Description: "Toggle feature groups.",
					Options: []discord.ApplicationCommandOption{
						discord.ApplicationCommandOptionBool{Name: "check_in_tools", Description: "Check-in tools"},
						discord.ApplicationCommandOptionBool{Name: "coping_tools", Description: "Coping tools"},
						discord.ApplicationCommandOptionBool{Name: "ghost_tools", Description: "Ghost letter tools"},
						discord.ApplicationCommandOptionBool{Name: "crisis_tools", Description: "Crisis tools"},
					},
				},
			},
			Run: runMellowAdmin,
		},
		{
			Name: "user", Description: "Manage users, roles, and bans.",
			Category: "Owner", Private: true, OwnerOnly: true, Cooldown: 3 * time.Second,
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionSubCommand{Name: "list", Description: "List recently seen users."},
				discord.ApplicationCommandOptionSubCommand{Name: "info", Description: "Show a user record.", Options: []discord.ApplicationCommandOption{userReq}},
				discord.ApplicationCommandOptionSubCommand{Name: "ban", Description: "Ban a user from Mellow.", Options: []discord.ApplicationCommandOption{userReq, reason}},
				discord.ApplicationCommandOptionSubCommand{Name: "unban", Description: "Unban a user.", Options: []discord.ApplicationCommandOption{userReq}},
				discord.ApplicationCommandOptionSubCommand{
					Name: "role", Description: "Set a user's role.",
					Options: []discord.ApplicationCommandOption{userReq, discord.ApplicationCommandOptionString{Name: "role", Description: "Role", Required: true, Choices: roleChoices}},
				},
				discord.ApplicationCommandOptionSubCommand{Name: "addowner", Description: "Add a Mellow owner.", Options: []discord.ApplicationCommandOption{userReq}},
				discord.ApplicationCommandOptionSubCommand{Name: "removeowner", Description: "Remove a Mellow owner.", Options: []discord.ApplicationCommandOption{userReq}},
			},
			Run: runUserAdmin,
		},
		{
			Name: "guild", Description: "Manage guilds Mellow is in.",
			Category: "Owner", Private: true, OwnerOnly: true, Cooldown: 3 * time.Second,
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionSubCommand{Name: "list", Description: "List recently joined guilds."},
				discord.ApplicationCommandOptionSubCommand{Name: "info", Description: "Show a guild record.", Options: []discord.ApplicationCommandOption{guildReq}},
				discord.ApplicationCommandOptionSubCommand{Name: "ban", Description: "Ban a guild.", Options: []discord.ApplicationCommandOption{guildReq, reason}},
				discord.ApplicationCommandOptionSubCommand{Name: "unban", Description: "Unban a guild.", Options: []discord.ApplicationCommandOption{guildReq}},
			},
			Run: runGuildAdmin,
		},
		{
			Name: "debug", Description: "Runtime diagnostics.",
			Category: "Owner", Private: true, OwnerOnly: true, Cooldown: 3 * time.Second,
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionSubCommand{Name: "database", Description: "Database status and counts."},
				discord.ApplicationCommandOptionSubCommand{Name: "ai", Description: "AI configuration and connectivity."},
				discord.ApplicationCommandOptionSubCommand{Name: "version", Description: "Build and runtime version."},
			},
			Run: runDebugAdmin,
		},
	}
}

func parseID(s string) (int64, bool) {
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return n, err == nil && n > 0
}

func runMellowAdmin(ctx context.Context, c *Ctx) error {
	switch c.Sub() {
	case "view":
		m, err := c.Store.RawMellow(ctx)
		if err != nil {
			return err
		}
		return c.Reply(infoEmbed("Mellow configuration", renderMellow(m)))
	case "toggle":
		m, err := c.Store.RawMellow(ctx)
		if err != nil {
			return err
		}
		if err := c.Store.SetMellowEnabled(ctx, !m.Enabled); err != nil {
			return err
		}
		c.AI.RefreshConfig(ctx)
		state := "enabled"
		if m.Enabled {
			state = "disabled"
		}
		return c.Reply(successEmbed("AI "+state, ""))
	case "reload":
		cfg := c.AI.RefreshConfig(ctx)
		return c.Reply(successEmbed("Configuration reloaded", "Model: "+cfg.Model))
	case "update":
		var upd db.MellowAIConfigUpdate
		if v := c.String("model"); v != "" {
			upd.Model = &v
		}
		if v := c.String("prompt"); v != "" {
			upd.Prompt = &v
		}
		if v, ok := c.Int("temperature_x10"); ok {
			if v < 0 || v > 20 {
				return c.ReplyEphemeral("temperature_x10 must be 0-20.")
			}
			f := float64(v) / 10
			upd.Temperature = &f
		}
		if v, ok := c.Int("max_tokens"); ok {
			if v < 1 || v > 4000 {
				return c.ReplyEphemeral("max_tokens must be 1-4000.")
			}
			n := int32(v)
			upd.MaxTokens = &n
		}
		if upd.Model == nil && upd.Prompt == nil && upd.Temperature == nil && upd.MaxTokens == nil {
			return c.ReplyEphemeral("Provide at least one field to update.")
		}
		m, err := c.Store.UpdateMellowAIConfig(ctx, upd)
		if err != nil {
			return err
		}
		c.AI.RefreshConfig(ctx)
		return c.Reply(successEmbed("Configuration updated", renderMellow(m)))
	case "features":
		checkIn := c.optBool("check_in_tools")
		coping := c.optBool("coping_tools")
		ghost := c.optBool("ghost_tools")
		crisis := c.optBool("crisis_tools")
		if checkIn == nil && coping == nil && ghost == nil && crisis == nil {
			return c.ReplyEphemeral("Provide at least one feature to toggle.")
		}
		m, err := c.Store.UpdateMellowFlags(ctx, checkIn, coping, ghost, crisis)
		if err != nil {
			return err
		}
		c.AI.RefreshConfig(ctx)
		return c.Reply(successEmbed("Features updated", renderMellow(m)))
	}
	return c.ReplyEphemeral("Unknown subcommand.")
}

func (c *Ctx) optBool(name string) *bool {
	if v, ok := c.Data.OptBool(name); ok {
		return &v
	}
	return nil
}

func renderMellow(m gen.Mellow) string {
	str := func(p *string) string {
		if p == nil {
			return "unset"
		}
		return *p
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Enabled: **%v**\n", m.Enabled)
	fmt.Fprintf(&b, "Model: **%s**\n", str(m.Model))
	fmt.Fprintf(&b, "Temperature: **%.2f** | Max tokens: **%d**\n", m.Temperature, m.MaxTokens)
	fmt.Fprintf(&b, "Features: check-in %v, coping %v, ghost %v, crisis %v\n",
		m.CheckInTools, m.CopingTools, m.GhostTools, m.CrisisTools)
	fmt.Fprintf(&b, "Owners: %d\n", len(m.Owners))
	prompt := str(m.Prompt)
	if len(prompt) > 500 {
		prompt = prompt[:500] + "..."
	}
	fmt.Fprintf(&b, "\nPrompt:\n%s\n\n(presence/frequency penalty are unused with Claude)", prompt)
	return b.String()
}

func runUserAdmin(ctx context.Context, c *Ctx) error {
	sub := c.Sub()

	if sub == "list" {
		rows, err := c.Store.ListRecentUsers(ctx, 20)
		if err != nil {
			return err
		}
		var b strings.Builder
		for _, u := range rows {
			fmt.Fprintf(&b, "`%d` %s - %s%s\n", u.ID, u.Username, db.UserRole(u), bannedTag(u.IsBanned))
		}
		return c.Reply(infoEmbed("Recent users", b.String()))
	}

	id, ok := parseID(c.String("user_id"))
	if !ok {
		return c.ReplyEphemeral("Invalid user ID.")
	}

	switch sub {
	case "info":
		u, err := c.Store.GetUser(ctx, id)
		if err != nil {
			return c.ReplyEphemeral("No record for that user.")
		}
		body := fmt.Sprintf("Username: %s\nRole: %s\nBanned: %v\nCreated: <t:%d:R>",
			u.Username, db.UserRole(u), u.IsBanned, u.CreatedAt.Unix())
		return c.Reply(infoEmbed("User "+strconv.FormatInt(id, 10), body))
	case "ban":
		var reason *string
		if r := c.String("reason"); r != "" {
			reason = &r
		}
		if _, err := c.Store.UpsertUser(ctx, id, "unknown"); err != nil {
			return err
		}
		if err := c.Store.SetUserBan(ctx, id, true, nil, reason); err != nil {
			return err
		}
		return c.Reply(successEmbed("User banned", strconv.FormatInt(id, 10)))
	case "unban":
		if err := c.Store.SetUserBan(ctx, id, false, nil, nil); err != nil {
			return err
		}
		return c.Reply(successEmbed("User unbanned", strconv.FormatInt(id, 10)))
	case "role":
		role := c.String("role")
		if _, err := c.Store.UpsertUser(ctx, id, "unknown"); err != nil {
			return err
		}
		if err := c.Store.SetUserRole(ctx, id, role); err != nil {
			return err
		}
		return c.Reply(successEmbed("Role set", fmt.Sprintf("%d is now %s", id, role)))
	case "addowner":
		if err := c.Store.AddMellowOwner(ctx, strconv.FormatInt(id, 10)); err != nil {
			return err
		}
		return c.Reply(successEmbed("Owner added", strconv.FormatInt(id, 10)))
	case "removeowner":
		if err := c.Store.RemoveMellowOwner(ctx, strconv.FormatInt(id, 10)); err != nil {
			return err
		}
		return c.Reply(successEmbed("Owner removed", strconv.FormatInt(id, 10)))
	}
	return c.ReplyEphemeral("Unknown subcommand.")
}

func runGuildAdmin(ctx context.Context, c *Ctx) error {
	sub := c.Sub()

	if sub == "list" {
		rows, err := c.Store.ListRecentGuilds(ctx, 20)
		if err != nil {
			return err
		}
		var b strings.Builder
		for _, g := range rows {
			fmt.Fprintf(&b, "`%d` %s%s\n", g.ID, g.Name, bannedTag(g.IsBanned))
		}
		return c.Reply(infoEmbed("Recent guilds", b.String()))
	}

	id, ok := parseID(c.String("guild_id"))
	if !ok {
		return c.ReplyEphemeral("Invalid guild ID.")
	}

	switch sub {
	case "info":
		g, err := c.Store.GetGuild(ctx, id)
		if err != nil {
			return c.ReplyEphemeral("No record for that guild.")
		}
		body := fmt.Sprintf("Name: %s\nOwner: `%d`\nBanned: %v\nJoined: <t:%d:R>",
			g.Name, g.OwnerId, g.IsBanned, g.JoinedAt.Unix())
		return c.Reply(infoEmbed("Guild "+strconv.FormatInt(id, 10), body))
	case "ban":
		var reason *string
		if r := c.String("reason"); r != "" {
			reason = &r
		}
		if err := c.Store.SetGuildBan(ctx, id, true, reason); err != nil {
			return err
		}
		return c.Reply(successEmbed("Guild banned", strconv.FormatInt(id, 10)))
	case "unban":
		if err := c.Store.SetGuildBan(ctx, id, false, nil); err != nil {
			return err
		}
		return c.Reply(successEmbed("Guild unbanned", strconv.FormatInt(id, 10)))
	}
	return c.ReplyEphemeral("Unknown subcommand.")
}

func runDebugAdmin(ctx context.Context, c *Ctx) error {
	switch c.Sub() {
	case "database":
		start := time.Now()
		err := c.Store.Ping(ctx)
		latency := time.Since(start)
		if err != nil {
			return c.Reply(errorEmbed("Database", err.Error()))
		}
		st, _ := c.Store.CommunityStats(ctx)
		body := fmt.Sprintf("Ping: %s\nDB rows - Users: %d | Guilds: %d | Conversations: %d | Crisis events: %d | Check-ins: %d\nLive cache - Users: %d | Guilds: %d",
			latency.Round(time.Millisecond), st.Users, st.Guilds, st.Conversations, st.CrisisEvents, st.MoodCheckIns,
			c.Bot.UserCount(), c.Bot.GuildCount())
		return c.Reply(infoEmbed("Database", body))
	case "ai":
		cfg := c.AI.Config(ctx)
		body := fmt.Sprintf("Live: %v\nEnabled: %v\nModel: %s\nTemperature: %.2f\nMax tokens: %d",
			c.AI.Live(), cfg.Enabled, cfg.Model, cfg.Temperature, cfg.MaxTokens)
		return c.Reply(infoEmbed("AI", body))
	case "version":
		return c.Reply(infoEmbed("Version", "mellow-go "+buildVersion))
	}
	return c.ReplyEphemeral("Unknown subcommand.")
}

func bannedTag(b bool) string {
	if b {
		return " (banned)"
	}
	return ""
}
