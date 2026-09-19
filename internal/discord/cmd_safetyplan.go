package discord

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/CodeMeAPixel/Mellow/internal/db"
	"github.com/CodeMeAPixel/Mellow/internal/helplines"
	"github.com/disgoorg/disgo/discord"
)

const maxPlanSection = 2000

type planSection struct {
	value string
	label string
	get   func(*db.SafetyPlan) *string
}

var planSections = []planSection{
	{"warning_signs", "Warning signs", func(p *db.SafetyPlan) *string { return &p.WarningSigns }},
	{"coping", "Things I can do on my own", func(p *db.SafetyPlan) *string { return &p.CopingStrategies }},
	{"distractions", "People and places that help me feel better", func(p *db.SafetyPlan) *string { return &p.Distractions }},
	{"people", "People I can ask for help", func(p *db.SafetyPlan) *string { return &p.PeopleToAsk }},
	{"professionals", "Professionals and services to contact", func(p *db.SafetyPlan) *string { return &p.Professionals }},
	{"safe_space", "Making my space safer", func(p *db.SafetyPlan) *string { return &p.SafeEnvironment }},
}

func (b *Bot) resourceBlock(ctx context.Context, userID int64) string {
	p, err := b.store.GetUserPreferences(ctx, userID)
	if err != nil || p.Country == nil || *p.Country == "" {
		return helplines.DefaultBlock
	}
	return helplines.Block(*p.Country)
}

func renderSafetyPlan(p db.SafetyPlan) string {
	var b strings.Builder
	for _, s := range planSections {
		v := strings.TrimSpace(*s.get(&p))
		if v == "" {
			continue
		}
		fmt.Fprintf(&b, "**%s**\n%s\n\n", s.label, v)
	}
	return strings.TrimSpace(b.String())
}

func (b *Bot) safetyPlanText(ctx context.Context, userID int64) string {
	p, err := b.store.GetSafetyPlan(ctx, userID)
	if err != nil || p.Empty() {
		return ""
	}
	return renderSafetyPlan(p)
}

func safetyPlanCommand() *Command {
	choices := make([]discord.ApplicationCommandOptionChoiceString, 0, len(planSections))
	for _, s := range planSections {
		choices = append(choices, discord.ApplicationCommandOptionChoiceString{Name: s.label, Value: s.value})
	}
	return &Command{
		Name: "safetyplan", Description: "Build a personal plan for hard moments. Only you can see it.",
		Category: "Crisis", Cooldown: 5 * time.Second,
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionSubCommand{Name: "view", Description: "Show your safety plan."},
			discord.ApplicationCommandOptionSubCommand{
				Name: "set", Description: "Write or replace one section of your plan.",
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionString{Name: "section", Description: "Which part of the plan", Required: true, Choices: choices},
					discord.ApplicationCommandOptionString{Name: "text", Description: "What to save in that section", Required: true},
				},
			},
			discord.ApplicationCommandOptionSubCommand{Name: "clear", Description: "Delete your entire safety plan."},
		},
		Run: runSafetyPlan,
	}
}

func runSafetyPlan(ctx context.Context, c *Ctx) error {
	switch c.Sub() {
	case "view":
		p, err := c.Store.GetSafetyPlan(ctx, c.UserID)
		if err != nil && !errors.Is(err, db.ErrNotFound) {
			return err
		}
		if errors.Is(err, db.ErrNotFound) || p.Empty() {
			return c.ReplyEphemeral("You have no safety plan yet. Use `/safetyplan set` to add a section, or build it on the dashboard.")
		}
		return c.ReplyEphemeral(renderSafetyPlan(p))

	case "set":
		section, text := c.String("section"), strings.TrimSpace(c.String("text"))
		if utf8.RuneCountInString(text) > maxPlanSection {
			return c.ReplyEphemeral("That is too long. Keep each section under 2000 characters.")
		}
		p, err := c.Store.GetSafetyPlan(ctx, c.UserID)
		if err != nil && !errors.Is(err, db.ErrNotFound) {
			return err
		}
		found := false
		for _, s := range planSections {
			if s.value == section {
				*s.get(&p) = text
				found = true
			}
		}
		if !found {
			return c.ReplyEphemeral("Unknown section.")
		}
		if err := c.Store.SaveSafetyPlan(ctx, c.UserID, p); err != nil {
			return err
		}
		return c.ReplyEphemeral("Saved. Only you can see your safety plan. Use `/safetyplan view` any time.")

	case "clear":
		if err := c.Store.DeleteSafetyPlan(ctx, c.UserID); err != nil {
			return err
		}
		return c.ReplyEphemeral("Your safety plan has been deleted.")
	}
	return c.ReplyEphemeral("Unknown subcommand.")
}
