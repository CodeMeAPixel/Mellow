package discord

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
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
	}
	return c.ReplyEphemeral("Unknown subcommand.")
}
