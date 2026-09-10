package discord

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/CodeMeAPixel/Mellow/internal/changelog"
	"github.com/disgoorg/disgo/discord"
)

func changelogCommands() []*Command {
	return []*Command{
		{
			Name: "changelog", Description: "Manage the bot listing changelog from PATCHNOTES.",
			Category: "Owner", Private: true, OwnerOnly: true, Cooldown: 3 * time.Second,
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionSubCommand{
					Name: "publish", Description: "Publish a PATCHNOTES entry to the listing.",
					Options: []discord.ApplicationCommandOption{
						discord.ApplicationCommandOptionString{Name: "version", Description: "Version to publish (default: latest)"},
						discord.ApplicationCommandOptionBool{Name: "force", Description: "Publish even if that version already exists"},
					},
				},
				discord.ApplicationCommandOptionSubCommand{Name: "list", Description: "List published changelog entries."},
				discord.ApplicationCommandOptionSubCommand{
					Name: "delete", Description: "Delete a published changelog entry.",
					Options: []discord.ApplicationCommandOption{
						discord.ApplicationCommandOptionString{Name: "id", Description: "Changelog entry ID", Required: true},
					},
				},
				discord.ApplicationCommandOptionSubCommand{Name: "preview", Description: "Show the latest PATCHNOTES entry without publishing."},
			},
			Run: runChangelog,
		},
	}
}

func runChangelog(ctx context.Context, c *Ctx) error {
	if c.Bot.omni == nil || !c.Bot.omni.Enabled() {
		if c.Sub() != "preview" {
			return c.ReplyEphemeral("Omniplex is not configured (OMNIPLEX_TOKEN / CLIENT_ID).")
		}
	}

	switch c.Sub() {
	case "preview":
		e, ok := changelog.Latest(changelog.Source())
		if !ok {
			return c.ReplyEphemeral("PATCHNOTES has no entries.")
		}
		return c.Reply(infoEmbed("v"+e.Version+" — "+e.Title, truncate(e.Content, 1800)))

	case "publish":
		_ = c.Defer(true)
		entries := changelog.Parse(changelog.Source())
		if len(entries) == 0 {
			return c.Reply(errorEmbed("Changelog", "PATCHNOTES has no entries."))
		}
		target := entries[0]
		if v := c.String("version"); v != "" {
			found := false
			for _, e := range entries {
				if strings.EqualFold(e.Version, strings.TrimPrefix(v, "v")) {
					target, found = e, true
					break
				}
			}
			if !found {
				return c.Reply(errorEmbed("Changelog", "No PATCHNOTES entry for version "+v+"."))
			}
		}

		if !c.Bool("force") {
			if existing, err := c.Bot.omni.GetChangelogs(ctx); err == nil {
				for _, ex := range existing {
					if strings.EqualFold(ex.Version, target.Version) {
						return c.Reply(infoEmbed("Changelog", "v"+target.Version+" is already published. Use force:true to publish again."))
					}
				}
			}
		}

		cl, err := c.Bot.omni.CreateChangelog(ctx, target.Version, target.Title, target.Content)
		if err != nil {
			return c.Reply(errorEmbed("Changelog publish failed", err.Error()))
		}
		return c.Reply(successEmbed("Published v"+target.Version, target.Title+"\n\nID: `"+cl.ID+"`"))

	case "list":
		_ = c.Defer(true)
		rows, err := c.Bot.omni.GetChangelogs(ctx)
		if err != nil {
			return c.Reply(errorEmbed("Changelog", err.Error()))
		}
		if len(rows) == 0 {
			return c.Reply(infoEmbed("Changelogs", "Nothing published yet."))
		}
		var b strings.Builder
		for _, r := range rows {
			fmt.Fprintf(&b, "`%s` **v%s** — %s\n", r.ID, r.Version, r.Title)
		}
		return c.Reply(infoEmbed("Published changelogs", b.String()))

	case "delete":
		_ = c.Defer(true)
		if err := c.Bot.omni.DeleteChangelog(ctx, c.String("id")); err != nil {
			return c.Reply(errorEmbed("Changelog", err.Error()))
		}
		return c.Reply(successEmbed("Changelog deleted", c.String("id")))
	}
	return c.ReplyEphemeral("Unknown subcommand.")
}
