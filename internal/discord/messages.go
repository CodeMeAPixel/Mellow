package discord

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/CodeMeAPixel/Mellow/internal/ai"
	"github.com/CodeMeAPixel/Mellow/internal/db"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

var mentionPrefix = regexp.MustCompile(`^<@!?\d+>\s*`)

func (b *Bot) onMessage(e *events.MessageCreate) {
	msg := e.Message
	if msg.Author.Bot || msg.Author.System {
		return
	}

	isDM := msg.GuildID == nil
	mentioned := false
	for _, u := range msg.Mentions {
		if u.ID == b.client.ID() {
			mentioned = true
			break
		}
	}
	if !isDM && !mentioned {
		return
	}

	content := strings.TrimSpace(mentionPrefix.ReplaceAllString(msg.Content, ""))
	if content == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	user := msg.Author
	userID := int64(user.ID)
	if _, err := b.store.UpsertUser(ctx, userID, user.Username); err != nil {
		return
	}

	var guildID string
	var guildIDNum *int64
	if msg.GuildID != nil {
		guildID = msg.GuildID.String()
		v := int64(*msg.GuildID)
		guildIDNum = &v
		if g, ok := e.Guild(); ok {
			_, _ = b.store.UpsertGuild(ctx, int64(g.ID), g.Name, int64(g.OwnerID))
		}
	}

	var personality, timezone, persona string
	crisisEnabled := true
	if prefs, perr := b.store.GetUserPreferences(ctx, userID); perr == nil {
		if prefs.AiPersonality != nil {
			personality = *prefs.AiPersonality
		}
		if prefs.Timezone != nil {
			timezone = *prefs.Timezone
		}
		crisisEnabled = !prefs.DisableCrisisDetection
		if prefs.CustomPersona != nil {
			persona = *prefs.CustomPersona
		}
	}

	var historyLen int32
	if has, _ := b.billing.HasPlusUser(ctx, userID); has {
		historyLen = plusHistoryLen
	} else {
		persona = ""
		if ai.IsPlusPersonality(personality) {
			personality = ""
		}
	}

	if crisisEnabled {
		if crisisReply, handled := b.handleConversationCrisis(ctx, content, userID, guildIDNum, guildID, e.ChannelID.String(), msg.ID.String()); handled {
			_, _ = b.client.Rest.CreateMessage(e.ChannelID, discord.MessageCreate{Content: crisisReply})
			return
		}
	}

	reply, err := b.ai.Generate(ctx, userID, content, ai.GenOpts{
		GuildID:       guildID,
		ChannelID:     e.ChannelID.String(),
		MessageID:     msg.ID.String(),
		IsDM:          isDM,
		Persist:       true,
		Personality:   personality,
		CustomPersona: persona,
		HistoryLen:    historyLen,
		Timezone:      timezone,
	})
	if err != nil {
		if errors.Is(err, ai.ErrNoKey) || errors.Is(err, ai.ErrDisabled) {
			return
		}
		_, _ = b.client.Rest.CreateMessage(e.ChannelID, discord.MessageCreate{
			Content: "I am having trouble responding right now. Please try again soon.",
		})
		return
	}

	_, _ = b.client.Rest.CreateMessage(e.ChannelID, discord.MessageCreate{Content: reply})
}

func (b *Bot) handleConversationCrisis(ctx context.Context, text string, userID int64, guildIDNum *int64, guildID, channelID, messageID string) (string, bool) {
	res, err := b.ai.AnalyzeCrisis(ctx, text)
	if err != nil || !res.NeedsSupport {
		return "", false
	}

	if !res.Respond {
		details := "[" + res.Level + "] " + res.Summary
		_, _ = b.store.CreateCrisisEvent(ctx, userID, &details, false)
		return "", false
	}

	reply := b.ai.CrisisResponseFor(ctx, res.Level, text, b.resourceBlock(ctx, userID))
	details := "[" + res.Level + "] bot reply: " + reply
	_, _ = b.store.CreateCrisisEvent(ctx, userID, &details, res.Level == "critical")
	b.syslog.Crisis(ctx, userID, guildIDNum, res.Level, res.Summary)
	if guildIDNum != nil {
		b.alertGuildCrisis(ctx, *guildIDNum, userID, res.Level, channelID, messageID)
	}

	var chp, gp, mp *string
	if channelID != "" {
		chp = &channelID
	}
	if guildID != "" {
		gp = &guildID
	}
	if messageID != "" {
		mp = &messageID
	}
	_, _ = b.store.AddConversationMessage(ctx, db.NewConversationMessage{UserID: userID, Content: text, IsAIResponse: false, ChannelID: chp, GuildID: gp, MessageID: mp})
	_, _ = b.store.AddConversationMessage(ctx, db.NewConversationMessage{UserID: userID, Content: reply, IsAIResponse: true, ChannelID: chp, GuildID: gp})
	return reply, true
}

func (b *Bot) handleCheckMessage(ctx context.Context, e *events.ApplicationCommandInteractionCreate, userID int64, guildID *int64) {
	data := e.MessageCommandInteractionData()
	target := data.TargetMessage()
	text := strings.TrimSpace(target.Content)
	if text == "" {
		_ = e.CreateMessage(discord.MessageCreate{
			Content: "That message has no readable text content.",
			Flags:   discord.MessageFlagEphemeral,
		})
		return
	}

	_ = e.DeferCreateMessage(true)

	res, err := b.ai.AnalyzeCrisis(ctx, text)
	if err != nil {
		_, _ = e.Client().Rest.CreateFollowupMessage(e.ApplicationID(), e.Token(), discord.MessageCreate{
			Content: "I could not analyse that message right now.",
			Flags:   discord.MessageFlagEphemeral,
		})
		return
	}

	if res.NeedsSupport {
		details := res.Summary
		_, _ = b.store.CreateCrisisEvent(ctx, int64(target.Author.ID), &details, res.Level == "critical")
		b.syslog.Crisis(ctx, int64(target.Author.ID), guildID, res.Level, res.Summary)
	}

	body := describeCrisis(res)
	if res.NeedsSupport {
		body += "\n\n" + b.resourceBlock(ctx, userID)
	}
	emb := infoEmbed("Message check", body)
	_, _ = e.Client().Rest.CreateFollowupMessage(e.ApplicationID(), e.Token(), discord.MessageCreate{
		Embeds: []discord.Embed{emb},
		Flags:  discord.MessageFlagEphemeral,
	})
}

func describeCrisis(res ai.CrisisResult) string {
	var b strings.Builder
	b.WriteString("Risk level: **" + res.Level + "**\n")
	if res.NeedsSupport {
		b.WriteString("This message may indicate someone needs support. A crisis event has been logged.\n")
	} else {
		b.WriteString("No immediate crisis indicators that warrant escalation.\n")
	}
	if res.Summary != "" {
		b.WriteString("\n" + res.Summary)
	}
	return b.String()
}
