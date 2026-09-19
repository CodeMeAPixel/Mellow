package discord

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
)

const (
	alertCooldown  = 10 * time.Minute
	maxExtraAlerts = 3
)

var alertSeen sync.Map

func alertRecipients(modAlert *string, extra []string, serverPlus bool) []string {
	seen := map[string]bool{}
	var out []string
	add := func(id string) {
		if id != "" && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	if modAlert != nil {
		add(*modAlert)
	}
	if serverPlus {
		added := 0
		for _, id := range extra {
			if added >= maxExtraAlerts {
				break
			}
			if id != "" && !seen[id] {
				add(id)
				added++
			}
		}
	}
	return out
}

func alertText(userID int64, level, guildID, channelID, messageID string) string {
	text := fmt.Sprintf("**Crisis alert (%s)**\n<@%d> may need support. Mellow has shared support resources with them.", level, userID)
	if channelID != "" && messageID != "" {
		text += fmt.Sprintf("\n[Jump to the message](https://discord.com/channels/%s/%s/%s)", guildID, channelID, messageID)
	}
	return text + "\nPlease reach out kindly and privately. The message text is not repeated here."
}

func (b *Bot) alertGuildCrisis(ctx context.Context, guildID, userID int64, level, channelID, messageID string) {
	g, err := b.store.GetGuild(ctx, guildID)
	if err != nil || !g.EnableCrisisAlerts {
		return
	}
	recipients := alertRecipients(g.ModAlertChannelId, g.ExtraAlertChannelIds, b.billing.HasPlusGuild(ctx, guildID))
	if len(recipients) == 0 {
		return
	}

	key := strconv.FormatInt(guildID, 10) + ":" + strconv.FormatInt(userID, 10)
	if last, ok := alertSeen.Load(key); ok && time.Since(last.(time.Time)) < alertCooldown {
		return
	}
	alertSeen.Store(key, time.Now())

	content := alertText(userID, level, strconv.FormatInt(guildID, 10), channelID, messageID)
	for _, id := range recipients {
		chID, err := snowflake.Parse(id)
		if err != nil {
			continue
		}
		if _, err := b.client.Rest.CreateMessage(chID, discord.MessageCreate{
			Content:         content,
			AllowedMentions: &discord.AllowedMentions{},
		}); err != nil {
			slog.Warn("crisis alert post failed", slog.String("channel", id), slog.String("err", err.Error()))
		}
	}
}
