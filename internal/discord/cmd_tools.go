package discord

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
)

func toolsCommands() []*Command {
	return []*Command{
		{
			Name: "tools", Description: "Operator diagnostics and repository status.",
			Category: "Owner", Private: true, OwnerOnly: true, Cooldown: 3 * time.Second,
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionSubCommand{Name: "status", Description: "Runtime status."},
				discord.ApplicationCommandOptionSubCommand{Name: "github", Description: "Repository info and latest release."},
				discord.ApplicationCommandOptionSubCommand{Name: "version", Description: "Build and runtime version."},
				discord.ApplicationCommandOptionSubCommand{
					Name: "logs", Description: "Recent internal logs.",
					Options: []discord.ApplicationCommandOption{
						discord.ApplicationCommandOptionString{
							Name: "type", Description: "Filter by type",
							Choices: []discord.ApplicationCommandOptionChoiceString{
								{Name: "gateway", Value: "gateway"},
								{Name: "guild", Value: "guild"},
								{Name: "command", Value: "command"},
								{Name: "crisis", Value: "crisis"},
								{Name: "system", Value: "system"},
								{Name: "startup", Value: "startup"},
								{Name: "shutdown", Value: "shutdown"},
							},
						},
					},
				},
				discord.ApplicationCommandOptionSubCommand{
					Name: "grantplus", Description: "Grant or revoke a test Mellow+ entitlement. No payment involved.",
					Options: []discord.ApplicationCommandOption{
						discord.ApplicationCommandOptionUser{Name: "user", Description: "User to grant or revoke for", Required: true},
						discord.ApplicationCommandOptionBool{Name: "revoke", Description: "Revoke instead of grant"},
					},
				},
			},
			Run: runTools,
		},
	}
}

func runTools(ctx context.Context, c *Ctx) error {
	switch c.Sub() {
	case "status":
		var mem runtime.MemStats
		runtime.ReadMemStats(&mem)
		up := time.Since(c.Bot.StartedAt()).Round(time.Second)
		dbErr := c.Store.Ping(ctx)
		dbState := "ok"
		if dbErr != nil {
			dbState = "error: " + dbErr.Error()
		}
		body := fmt.Sprintf(
			"Uptime: %s\nGoroutines: %d\nHeap: %.1f MiB\nDatabase: %s\nAI: live=%v",
			up, runtime.NumGoroutine(), float64(mem.HeapAlloc)/(1024*1024), dbState, c.AI.Live(),
		)
		return c.Reply(infoEmbed("Runtime status", body))
	case "github":
		_ = c.Defer(true)
		gh := c.Bot.gh
		info, err := gh.RepoInfo(ctx)
		if err != nil {
			return c.Reply(errorEmbed("GitHub", err.Error()))
		}
		emb := infoEmbed(info.FullName, info.Description).
			AddField("Stars", fmt.Sprintf("%d", info.Stars), true).
			AddField("Forks", fmt.Sprintf("%d", info.Forks), true).
			AddField("Open issues", fmt.Sprintf("%d", info.OpenIssues), true)
		if rel, err := gh.LatestRelease(ctx); err == nil && rel.TagName != "" {
			notes := strings.TrimSpace(rel.Body)
			if len(notes) > 400 {
				notes = notes[:400] + "..."
			}
			emb = emb.AddField("Latest release", rel.TagName+"\n"+notes, false)
		}
		return c.Reply(emb)
	case "version":
		return c.Reply(infoEmbed("Version", "mellow-go "+buildVersion+"\nGo "+runtime.Version()+"\n"+sourceLink()))
	case "logs":
		var lt *string
		if v := c.String("type"); v != "" {
			lt = &v
		}
		rows, err := c.Store.RecentSystemLogs(ctx, lt, 15)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return c.Reply(infoEmbed("Internal logs", "Nothing recorded."))
		}
		var b strings.Builder
		for _, r := range rows {
			line := fmt.Sprintf("`%s` **%s** — %s <t:%d:R>", r.LogType, r.Severity, r.Title, r.CreatedAt.Unix())
			if r.Description != nil && *r.Description != "" {
				d := *r.Description
				if len(d) > 120 {
					d = d[:120] + "..."
				}
				line += "\n" + d
			}
			b.WriteString(line + "\n")
		}
		return c.Reply(infoEmbed("Recent internal logs", b.String()))
	case "grantplus":
		return runGrantPlus(ctx, c)
	}
	return c.ReplyEphemeral("Unknown subcommand.")
}

func runGrantPlus(ctx context.Context, c *Ctx) error {
	if !c.Bot.billing.Enabled() {
		return c.ReplyEphemeral("SKU_MELLOW_PLUS_ID isn't set, so there's no SKU to grant.")
	}
	target, ok := c.Data.OptUser("user")
	if !ok {
		return c.ReplyEphemeral("Specify a user.")
	}
	appID := c.Bot.client.ApplicationID
	skuID := c.Bot.billing.PlusSKU()

	if c.Bool("revoke") {
		ent, err := c.Store.ActiveUserEntitlement(ctx, int64(target.ID), int64(skuID))
		if err != nil {
			return c.ReplyEphemeral(target.Username + " has no active Mellow+ entitlement to revoke.")
		}
		if err := c.Bot.client.Rest.DeleteTestEntitlement(appID, snowflake.ID(ent.ID)); err != nil {
			return c.Reply(errorEmbed("Revoke failed", err.Error()))
		}
		_ = c.Bot.billing.Remove(ctx, snowflake.ID(ent.ID))
		return c.Reply(successEmbed("Test entitlement revoked", "Mellow+ (test) revoked from "+target.Username+"."))
	}

	ent, err := c.Bot.client.Rest.CreateTestEntitlement(appID, discord.TestEntitlementCreate{
		SkuID:     skuID,
		OwnerID:   target.ID,
		OwnerType: discord.EntitlementOwnerTypeUser,
	})
	if err != nil {
		return c.Reply(errorEmbed("Grant failed", err.Error()))
	}
	_ = c.Bot.billing.Sync(ctx, *ent)
	return c.Reply(successEmbed("Test entitlement granted", "Mellow+ (test) granted to "+target.Username+". Use `/tools grantplus revoke:true` to remove it."))
}
