package discord

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CodeMeAPixel/Mellow/internal/ai"
	"github.com/disgoorg/disgo/discord"
)

var aiCopingTools = []string{"breathing", "grounding", "affirmations", "challenge", "distraction", "music"}

func copingCommand() *Command {
	feeling := discord.ApplicationCommandOptionString{Name: "feeling", Description: "How are you feeling? (optional)"}

	opts := []discord.ApplicationCommandOption{}
	for _, tool := range aiCopingTools {
		opts = append(opts, discord.ApplicationCommandOptionSubCommand{
			Name: tool, Description: "Get a " + tool + " coping exercise.",
			Options: []discord.ApplicationCommandOption{feeling},
		})
	}
	opts = append(opts,
		discord.ApplicationCommandOptionSubCommand{
			Name: "gratitude", Description: "Log or view things you are grateful for.",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionString{Name: "item", Description: "Something you are grateful for"},
			},
		},
		discord.ApplicationCommandOptionSubCommand{
			Name: "journal", Description: "Write, view, or delete journal entries.",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionString{Name: "write", Description: "Text to save as a new entry"},
				discord.ApplicationCommandOptionInt{Name: "delete", Description: "Entry number to delete (from the list)"},
				discord.ApplicationCommandOptionString{Name: "search", Description: "Plus: find entries containing this text"},
				discord.ApplicationCommandOptionString{Name: "tag", Description: "Plus: find entries with this #tag (add #tags when you write)"},
			},
		},
		discord.ApplicationCommandOptionSubCommand{
			Name: "plan", Description: "View or set your personalised coping plan.",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionString{Name: "set", Description: "Replace your coping plan with this text"},
				discord.ApplicationCommandOptionBool{Name: "clear", Description: "Clear your coping plan"},
			},
		},
		discord.ApplicationCommandOptionSubCommand{
			Name: "streaks", Description: "View your coping tool usage streaks.",
		},
		discord.ApplicationCommandOptionSubCommand{
			Name: "toolbox", Description: "Manage your favourite coping tools.",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionString{Name: "add", Description: "Add a favourite tool"},
				discord.ApplicationCommandOptionString{Name: "remove", Description: "Remove a favourite tool"},
			},
		},
	)

	return &Command{
		Name: "coping", Description: "Coping tools and exercises for your wellbeing.",
		Category: "Coping", Cooldown: 6 * time.Second, PremiumCooldown: 3 * time.Second,
		Options: opts,
		Run:     runCoping,
	}
}

func runCoping(ctx context.Context, c *Ctx) error {
	sub := c.Sub()
	for _, tool := range aiCopingTools {
		if sub == tool {
			return runAICoping(ctx, c, tool)
		}
	}
	switch sub {
	case "gratitude":
		return runGratitude(ctx, c)
	case "journal":
		return runJournal(ctx, c)
	case "plan":
		return runPlan(ctx, c)
	case "streaks":
		return runStreaks(ctx, c)
	case "toolbox":
		return runToolbox(ctx, c)
	}
	return c.ReplyEphemeral("Unknown coping subcommand.")
}

func runAICoping(ctx context.Context, c *Ctx, tool string) error {
	_ = c.Defer(false)
	_ = c.Store.RecordCopingToolUsageIn(ctx, c.UserID, tool, c.Bot.activityGuild(c.GuildID))

	out, err := c.AI.Coping(ctx, tool, c.String("feeling"))
	if err != nil {
		if errors.Is(err, ai.ErrNoKey) {
			return c.Reply(infoEmbed(strings.Title(tool), staticCoping[tool]))
		}
		return err
	}
	return c.Reply(infoEmbed(strings.Title(tool)+" exercise", out))
}

var staticCoping = map[string]string{
	"breathing":    "Breathe in for 4 counts, hold for 7, breathe out for 8. Repeat four times.",
	"grounding":    "Name 5 things you can see, 4 you can touch, 3 you can hear, 2 you can smell, 1 you can taste.",
	"affirmations": "You are doing your best, and that is enough. This feeling will pass. You deserve care.",
	"challenge":    "Drink a glass of water and step outside for two minutes of fresh air.",
	"distraction":  "Try counting backwards from 100 by 7s, or name an animal for every letter of the alphabet.",
	"music":        "Look for calm playlists such as lo-fi, ambient, or gentle piano to help settle your mind.",
}

func runGratitude(ctx context.Context, c *Ctx) error {
	if item := c.String("item"); item != "" {
		if _, err := c.Store.CreateGratitudeEntry(ctx, c.UserID, item); err != nil {
			return err
		}
		_ = c.Store.RecordCopingToolUsage(ctx, c.UserID, "gratitude")
		return c.Reply(successEmbed("Gratitude logged", "Saved: "+item))
	}
	rows, err := c.Store.GratitudeEntriesForUser(ctx, c.UserID, 10)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return c.Reply(infoEmbed("Gratitude", "Nothing logged yet. Add one with /coping gratitude item:..."))
	}
	var b strings.Builder
	for _, r := range rows {
		fmt.Fprintf(&b, "- %s  <t:%d:R>\n", r.Item, r.CreatedAt.Unix())
	}
	return c.Reply(infoEmbed("Things you are grateful for", b.String()))
}

func runJournal(ctx context.Context, c *Ctx) error {
	if text := c.String("write"); text != "" {
		if _, err := c.Store.CreateJournalEntry(ctx, c.UserID, text, true); err != nil {
			return err
		}
		_ = c.Store.RecordCopingToolUsage(ctx, c.UserID, "journal")
		return c.ReplyEphemeral("Journal entry saved.")
	}
	if n, ok := c.Int("delete"); ok {
		rows, err := c.Store.JournalEntriesForUser(ctx, c.UserID, 25, 0)
		if err != nil {
			return err
		}
		if n < 1 || n > len(rows) {
			return c.ReplyEphemeral("That entry number is out of range.")
		}
		if _, err := c.Store.DeleteJournalEntry(ctx, rows[n-1].ID, c.UserID); err != nil {
			return err
		}
		return c.ReplyEphemeral(fmt.Sprintf("Deleted entry %d.", n))
	}
	if search, tag := c.String("search"), c.String("tag"); search != "" || tag != "" {
		if !c.HasPlus(ctx) {
			return c.ReplyEphemeral(plusOnly("Searching your journal"))
		}
		all, err := c.Store.JournalEntriesForUser(ctx, c.UserID, 500, 0)
		if err != nil {
			return err
		}
		matches := filterJournal(all, search, tag)
		if len(matches) == 0 {
			return c.ReplyEphemeral("No journal entries matched.")
		}
		return c.ReplyEphemeral(renderJournalMatches(matches, 8))
	}
	rows, err := c.Store.JournalEntriesForUser(ctx, c.UserID, 10, 0)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return c.ReplyEphemeral("No journal entries yet. Add one with /coping journal write:...")
	}
	var b strings.Builder
	for i, r := range rows {
		body := r.Content
		if len(body) > 200 {
			body = body[:200] + "..."
		}
		fmt.Fprintf(&b, "**%d.** <t:%d:R>\n%s\n\n", i+1, r.CreatedAt.Unix(), body)
	}
	return c.ReplyEphemeral(b.String())
}

func runPlan(ctx context.Context, c *Ctx) error {
	if c.Bool("clear") {
		plan, err := c.Store.GetCopingPlan(ctx, c.UserID)
		if err != nil {
			return c.ReplyEphemeral("You have no coping plan to clear.")
		}
		if _, err := c.Store.DeleteCopingPlan(ctx, plan.ID, c.UserID); err != nil {
			return err
		}
		return c.ReplyEphemeral("Coping plan cleared.")
	}
	if text := c.String("set"); text != "" {
		existing, err := c.Store.GetCopingPlan(ctx, c.UserID)
		if err != nil {
			if _, cerr := c.Store.CreateCopingPlan(ctx, c.UserID, text); cerr != nil {
				return cerr
			}
		} else if _, uerr := c.Store.UpdateCopingPlan(ctx, existing.ID, text); uerr != nil {
			return uerr
		}
		_ = c.Store.RecordCopingToolUsage(ctx, c.UserID, "plan")
		return c.Reply(successEmbed("Coping plan saved", text))
	}
	plan, err := c.Store.GetCopingPlan(ctx, c.UserID)
	if err != nil {
		return c.Reply(infoEmbed("Coping plan", "You have not set a coping plan yet. Set one with /coping plan set:..."))
	}
	return c.Reply(infoEmbed("Your coping plan", plan.Plan))
}

func runStreaks(ctx context.Context, c *Ctx) error {
	rows, err := c.Store.CopingToolUsagesForUser(ctx, c.UserID, 500)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return c.Reply(infoEmbed("Streaks", "No coping tool usage recorded yet."))
	}
	days := map[string]bool{}
	for _, r := range rows {
		days[r.UsedAt.Format("2006-01-02")] = true
	}
	streak := 0
	day := time.Now()
	for days[day.Format("2006-01-02")] || (streak == 0 && day.Format("2006-01-02") == time.Now().Format("2006-01-02")) {
		if days[day.Format("2006-01-02")] {
			streak++
		}
		day = day.AddDate(0, 0, -1)
		if streak > 3650 {
			break
		}
	}
	counts, _ := c.Store.CopingToolUsageCounts(ctx, c.UserID)
	var b strings.Builder
	fmt.Fprintf(&b, "Current daily streak: **%d**\nTotal uses: **%d**\n\n", streak, len(rows))
	for _, cnt := range counts {
		fmt.Fprintf(&b, "%s x%d\n", cnt.ToolName, cnt.Uses)
	}
	return c.Reply(infoEmbed("Coping streaks", b.String()))
}

func runToolbox(ctx context.Context, c *Ctx) error {
	if add := c.String("add"); add != "" {
		if err := c.Store.AddFavoriteCopingTool(ctx, c.UserID, add); err != nil {
			return err
		}
		return c.ReplyEphemeral("Added " + add + " to your toolbox.")
	}
	if rm := c.String("remove"); rm != "" {
		if _, err := c.Store.RemoveFavoriteCopingTool(ctx, c.UserID, rm); err != nil {
			return err
		}
		return c.ReplyEphemeral("Removed " + rm + " from your toolbox.")
	}
	rows, err := c.Store.FavoriteCopingTools(ctx, c.UserID)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return c.Reply(infoEmbed("Your toolbox", "Empty. Add tools with /coping toolbox add:..."))
	}
	names := make([]string, 0, len(rows))
	for _, r := range rows {
		names = append(names, r.Tool)
	}
	return c.Reply(infoEmbed("Your toolbox", strings.Join(names, ", ")))
}
