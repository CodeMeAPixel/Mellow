package discord

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"
)

func infoCommands() []*Command {
	return []*Command{
		{
			Name: "ping", Description: "Check the bot's latency and uptime.", Category: "Info",
			Cooldown: 5 * time.Second,
			Run: func(ctx context.Context, c *Ctx) error {
				up := time.Since(c.Bot.StartedAt()).Round(time.Second)
				emb := infoEmbed("Pong", fmt.Sprintf("Uptime: %s\nRestarts: %d\nGoroutines: %d", up, c.Bot.RestartCount(), runtime.NumGoroutine()))
				return c.Reply(emb)
			},
		},
		{
			Name: "about", Description: "Learn about Mellow and our mission.", Category: "Info",
			Run: func(ctx context.Context, c *Ctx) error {
				return c.Reply(infoEmbed("About Mellow",
					"Mellow is an AI mental health companion for Discord. It offers check-ins, coping tools, crisis support resources, and a listening ear.\n\n"+
						"Built by "+authorName+" ("+authorURL+")\nSource: "+sourceLink()+"\nLicensed under AGPL-3.0.\n\n"+webBase()))
			},
		},
		{
			Name: "invite", Description: "Get the invite link for Mellow.", Category: "Info",
			Run: func(ctx context.Context, c *Ctx) error {
				return c.Reply(infoEmbed("Invite Mellow", inviteLink()))
			},
		},
		{
			Name: "support", Description: "Get help with Mellow or join our support community.", Category: "Info",
			Run: func(ctx context.Context, c *Ctx) error {
				return c.Reply(infoEmbed("Support", "Need help? Visit "+supportLink()))
			},
		},
		{
			Name: "privacy", Description: "Learn about how your data is handled.", Category: "Info",
			Run: func(ctx context.Context, c *Ctx) error {
				return c.Reply(infoEmbed("Privacy",
					"Your conversations and journal entries are stored encrypted at rest. You can opt out of context logging and crisis detection in your preferences.\n\n"+webBase()+"/privacy"))
			},
		},
		{
			Name: "docs", Description: "Get the link to the full Mellow documentation.", Category: "Info",
			Run: func(ctx context.Context, c *Ctx) error {
				return c.Reply(infoEmbed("Documentation", docsLink()))
			},
		},
		{
			Name: "source", Description: "Mellow is open source, get the repo link here.", Category: "Info",
			Run: func(ctx context.Context, c *Ctx) error {
				return c.Reply(infoEmbed("Source",
					sourceLink()+"\n\nBuilt by "+authorName+" ("+authorURL+"), licensed under AGPL-3.0."))
			},
		},
		{
			Name: "version", Description: "View Mellow version information.", Category: "Info",
			Cooldown: 10 * time.Second,
			Run: func(ctx context.Context, c *Ctx) error {
				return c.Reply(infoEmbed("Version", c.Bot.versionText(ctx)))
			},
		},
		{
			Name: "stats", Description: "View Mellow community statistics.", Category: "Info",
			Cooldown: 10 * time.Second,
			Run: func(ctx context.Context, c *Ctx) error {
				st, err := c.Store.CommunityStats(ctx)
				if err != nil {
					return err
				}
				emb := infoEmbed("Community stats", "").
					AddField("Users", fmt.Sprintf("%d", c.Bot.UserCount()), true).
					AddField("Servers", fmt.Sprintf("%d", c.Bot.GuildCount()), true).
					AddField("Conversations", fmt.Sprintf("%d", st.Conversations), true).
					AddField("Check-ins", fmt.Sprintf("%d", st.MoodCheckIns), true).
					AddField("Crisis events", fmt.Sprintf("%d", st.CrisisEvents), true)
				return c.Reply(emb)
			},
		},
		{
			Name: "help", Description: "Shows all Mellow commands.", Category: "Info",
			Run: func(ctx context.Context, c *Ctx) error {
				emb := infoEmbed("Mellow commands", c.Bot.helpText())
				return c.Reply(emb)
			},
		},
	}
}

var buildVersion = "dev"

func SetVersion(v string) {
	if v != "" {
		buildVersion = v
	}
}

func Version() string { return buildVersion }

func runningRelease(running, tag string) bool {
	running = strings.TrimPrefix(running, "v")
	tag = strings.TrimPrefix(tag, "v")
	return running == tag || strings.HasPrefix(running, tag+"-")
}

func (b *Bot) CheckForUpdates(ctx context.Context) {
	rel, err := b.gh.LatestRelease(ctx)
	if err != nil || rel.TagName == "" {
		return
	}
	if buildVersion == "dev" || runningRelease(buildVersion, rel.TagName) {
		return
	}
	b.syslog.Event(ctx, "system", "Update available",
		"Running "+buildVersion+", latest release is "+rel.TagName+" ("+rel.HTMLURL+")", "warning")
}

func (b *Bot) versionText(ctx context.Context) string {
	lines := []string{"Running: **" + buildVersion + "**", "Go " + runtime.Version()}
	if rel, err := b.gh.LatestRelease(ctx); err == nil && rel.TagName != "" {
		if runningRelease(buildVersion, rel.TagName) {
			lines = append(lines, "Up to date with the latest release ("+rel.TagName+").")
		} else {
			lines = append(lines, "Latest release: **"+rel.TagName+"** — "+rel.HTMLURL)
		}
	}
	lines = append(lines, sourceLink())
	return strings.Join(lines, "\n")
}

func (b *Bot) helpText() string {
	byCat := map[string][]string{}
	for _, cmd := range b.commands {
		if cmd.Category == "Stub" || cmd.Private {
			continue
		}
		byCat[cmd.Category] = append(byCat[cmd.Category], cmd.Name)
	}
	var out string
	for _, cat := range []string{"Info", "Check-In", "Coping", "Crisis", "Users", "Fun", "Guild"} {
		names := byCat[cat]
		if len(names) == 0 {
			continue
		}
		out += "**" + cat + "**\n"
		for _, n := range names {
			out += "/" + n + "  "
		}
		out += "\n\n"
	}
	return out
}
