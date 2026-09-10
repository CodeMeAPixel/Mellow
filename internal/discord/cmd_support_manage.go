package discord

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
)

func supportManageCommands() []*Command {
	idReq := discord.ApplicationCommandOptionInt{Name: "id", Description: "Record ID", Required: true}
	limitOpt := discord.ApplicationCommandOptionInt{Name: "limit", Description: "How many to list (max 25)"}
	statusChoices := []discord.ApplicationCommandOptionChoiceString{
		{Name: "open", Value: "open"},
		{Name: "investigating", Value: "investigating"},
		{Name: "resolved", Value: "resolved"},
		{Name: "closed", Value: "closed"},
	}

	return []*Command{
		{
			Name: "feedback-manage", Description: "Review and moderate user feedback.",
			Category: "Admin", Private: true, OwnerOnly: true, Cooldown: 3 * time.Second,
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionSubCommand{
					Name: "list", Description: "List feedback.",
					Options: []discord.ApplicationCommandOption{
						discord.ApplicationCommandOptionBool{Name: "approved_only", Description: "Only approved feedback"},
						limitOpt,
					},
				},
				discord.ApplicationCommandOptionSubCommand{Name: "view", Description: "View one feedback item and its replies.", Options: []discord.ApplicationCommandOption{idReq}},
				discord.ApplicationCommandOptionSubCommand{
					Name: "reply", Description: "Reply to a feedback item.",
					Options: []discord.ApplicationCommandOption{idReq, discord.ApplicationCommandOptionString{Name: "message", Description: "Reply text", Required: true}},
				},
				discord.ApplicationCommandOptionSubCommand{
					Name: "approve", Description: "Approve feedback, optionally publish or feature it.",
					Options: []discord.ApplicationCommandOption{
						idReq,
						discord.ApplicationCommandOptionBool{Name: "public", Description: "Make it public"},
						discord.ApplicationCommandOptionBool{Name: "featured", Description: "Feature it"},
					},
				},
				discord.ApplicationCommandOptionSubCommand{Name: "unapprove", Description: "Unapprove feedback.", Options: []discord.ApplicationCommandOption{idReq}},
				discord.ApplicationCommandOptionSubCommand{Name: "delete", Description: "Delete feedback.", Options: []discord.ApplicationCommandOption{idReq}},
			},
			Run: runFeedbackManage,
		},
		{
			Name: "report-manage", Description: "Review and triage user reports.",
			Category: "Admin", Private: true, OwnerOnly: true, Cooldown: 3 * time.Second,
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionSubCommand{
					Name: "list", Description: "List reports.",
					Options: []discord.ApplicationCommandOption{
						discord.ApplicationCommandOptionString{Name: "status", Description: "Filter by status", Choices: statusChoices},
						limitOpt,
					},
				},
				discord.ApplicationCommandOptionSubCommand{Name: "view", Description: "View one report and its replies.", Options: []discord.ApplicationCommandOption{idReq}},
				discord.ApplicationCommandOptionSubCommand{
					Name: "status", Description: "Set a report's status.",
					Options: []discord.ApplicationCommandOption{idReq, discord.ApplicationCommandOptionString{Name: "status", Description: "New status", Required: true, Choices: statusChoices}},
				},
				discord.ApplicationCommandOptionSubCommand{
					Name: "reply", Description: "Reply to a report.",
					Options: []discord.ApplicationCommandOption{idReq, discord.ApplicationCommandOptionString{Name: "message", Description: "Reply text", Required: true}},
				},
			},
			Run: runReportManage,
		},
	}
}

func listLimit(c *Ctx) int32 {
	if v, ok := c.Int("limit"); ok && v > 0 && v <= 25 {
		return int32(v)
	}
	return 10
}

func runFeedbackManage(ctx context.Context, c *Ctx) error {
	switch c.Sub() {
	case "list":
		var approved *bool
		if v := c.Bool("approved_only"); v {
			approved = &v
		}
		rows, err := c.Store.ListFeedback(ctx, listLimit(c), approved)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return c.Reply(infoEmbed("Feedback", "No feedback found."))
		}
		var b strings.Builder
		for _, f := range rows {
			flags := []string{}
			if f.Approved {
				flags = append(flags, "approved")
			}
			if f.Public {
				flags = append(flags, "public")
			}
			if f.Featured {
				flags = append(flags, "featured")
			}
			fmt.Fprintf(&b, "`#%d` <t:%d:R> %s\n%s\n\n", f.ID, f.CreatedAt.Unix(), strings.Join(flags, " "), truncate(f.Message, 160))
		}
		return c.Reply(infoEmbed("Feedback", b.String()))
	case "view":
		id := int32(mustInt(c, "id"))
		f, err := c.Store.GetFeedback(ctx, id)
		if err != nil {
			return c.ReplyEphemeral("No feedback with that ID.")
		}
		replies, _ := c.Store.FeedbackReplies(ctx, id)
		var b strings.Builder
		fmt.Fprintf(&b, "Submitted <t:%d:R>\napproved=%v public=%v featured=%v\n\n%s\n", f.CreatedAt.Unix(), f.Approved, f.Public, f.Featured, f.Message)
		for _, r := range replies {
			fmt.Fprintf(&b, "\nStaff reply <t:%d:R>:\n%s\n", r.CreatedAt.Unix(), r.Message)
		}
		return c.Reply(infoEmbed(fmt.Sprintf("Feedback #%d", id), b.String()))
	case "reply":
		id := int32(mustInt(c, "id"))
		if err := c.Store.CreateFeedbackReply(ctx, id, c.UserID, c.String("message")); err != nil {
			return err
		}
		return c.Reply(successEmbed("Reply saved", fmt.Sprintf("Added a reply to feedback #%d", id)))
	case "approve":
		id := int32(mustInt(c, "id"))
		approved := true
		pub := c.optBool("public")
		feat := c.optBool("featured")
		if _, err := c.Store.SetFeedbackApproval(ctx, id, &approved, pub, feat); err != nil {
			return err
		}
		return c.Reply(successEmbed("Feedback approved", fmt.Sprintf("#%d", id)))
	case "unapprove":
		id := int32(mustInt(c, "id"))
		no := false
		if _, err := c.Store.SetFeedbackApproval(ctx, id, &no, &no, &no); err != nil {
			return err
		}
		return c.Reply(successEmbed("Feedback unapproved", fmt.Sprintf("#%d", id)))
	case "delete":
		id := int32(mustInt(c, "id"))
		n, err := c.Store.DeleteFeedback(ctx, id)
		if err != nil {
			return err
		}
		if n == 0 {
			return c.ReplyEphemeral("No feedback with that ID.")
		}
		return c.Reply(successEmbed("Feedback deleted", fmt.Sprintf("#%d", id)))
	}
	return c.ReplyEphemeral("Unknown subcommand.")
}

func runReportManage(ctx context.Context, c *Ctx) error {
	switch c.Sub() {
	case "list":
		var status *string
		if v := c.String("status"); v != "" {
			status = &v
		}
		rows, err := c.Store.ListReports(ctx, listLimit(c), status)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return c.Reply(infoEmbed("Reports", "No reports found."))
		}
		var b strings.Builder
		for _, r := range rows {
			fmt.Fprintf(&b, "`#%d` [%s] <t:%d:R>\n%s\n\n", r.ID, r.Status, r.CreatedAt.Unix(), truncate(r.Message, 160))
		}
		return c.Reply(infoEmbed("Reports", b.String()))
	case "view":
		id := int32(mustInt(c, "id"))
		r, err := c.Store.GetReport(ctx, id)
		if err != nil {
			return c.ReplyEphemeral("No report with that ID.")
		}
		replies, _ := c.Store.ReportReplies(ctx, id)
		var b strings.Builder
		fmt.Fprintf(&b, "Status: **%s**\nSubmitted <t:%d:R>\n\n%s\n", r.Status, r.CreatedAt.Unix(), r.Message)
		for _, rr := range replies {
			fmt.Fprintf(&b, "\nStaff reply <t:%d:R>:\n%s\n", rr.CreatedAt.Unix(), rr.Message)
		}
		return c.Reply(infoEmbed(fmt.Sprintf("Report #%d", id), b.String()))
	case "status":
		id := int32(mustInt(c, "id"))
		if _, err := c.Store.SetReportStatus(ctx, id, c.String("status")); err != nil {
			return err
		}
		return c.Reply(successEmbed("Report updated", fmt.Sprintf("#%d is now %s", id, c.String("status"))))
	case "reply":
		id := int32(mustInt(c, "id"))
		if err := c.Store.CreateReportReply(ctx, id, c.UserID, c.String("message")); err != nil {
			return err
		}
		return c.Reply(successEmbed("Reply saved", fmt.Sprintf("Added a reply to report #%d", id)))
	}
	return c.ReplyEphemeral("Unknown subcommand.")
}

func mustInt(c *Ctx, name string) int {
	v, _ := c.Int(name)
	return v
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
