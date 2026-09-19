package discord

import (
	"context"
	"log/slog"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func (b *Bot) onEntitlementCreate(e *events.EntitlementCreate) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := b.billing.Sync(ctx, e.Entitlement); err != nil {
		slog.Warn("entitlement sync failed", slog.String("err", err.Error()))
	}
}

func (b *Bot) onEntitlementUpdate(e *events.EntitlementUpdate) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := b.billing.Sync(ctx, e.Entitlement); err != nil {
		slog.Warn("entitlement sync failed", slog.String("err", err.Error()))
	}
}

func (b *Bot) onEntitlementDelete(e *events.EntitlementDelete) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := b.billing.Remove(ctx, e.ID); err != nil {
		slog.Warn("entitlement removal failed", slog.String("err", err.Error()))
	}
}

func (b *Bot) ReconcileEntitlements(ctx context.Context) {
	if !b.billing.Enabled() && !b.billing.ServerEnabled() {
		return
	}
	appID := b.client.ApplicationID
	if appID == 0 {
		return
	}
	n, err := b.billing.Reconcile(ctx, b.client.Rest, appID)
	if err != nil {
		slog.Warn("entitlement reconcile failed", slog.String("err", err.Error()))
		return
	}
	slog.Info("entitlements reconciled", slog.Int("count", n))
}

func (b *Bot) RunEntitlementSync(ctx context.Context) {
	if !b.billing.Enabled() && !b.billing.ServerEnabled() {
		return
	}
	select {
	case <-ctx.Done():
		return
	case <-time.After(20 * time.Second):
	}
	b.ReconcileEntitlements(ctx)

	t := time.NewTicker(30 * time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			b.ReconcileEntitlements(ctx)
		}
	}
}

func upgradeCommand() *Command {
	return &Command{
		Name: "upgrade", Description: "See what Mellow+ offers and get it.", Category: "Info",
		Run: func(ctx context.Context, c *Ctx) error {
			if !c.Bot.billing.Enabled() {
				return c.ReplyEphemeral("Mellow+ isn't available yet.")
			}
			has := c.HasPlus(ctx)
			desc := "Mellow+ is an optional way to support Mellow and get a few extras. " +
				"It never changes what safety features are available to anyone.\n\n" +
				"- Longer `/insights` history instead of the last 30 days\n" +
				"- Shorter cooldowns on `/coping` and fun commands\n" +
				"- A supporter flair on `/profile`"
			if has {
				desc += "\n\nYou already have Mellow+. Thank you for supporting Mellow."
			}
			emb := infoEmbed("Mellow+", desc)
			msg := discord.MessageCreate{Embeds: []discord.Embed{emb}}
			if !has {
				msg.Components = []discord.LayoutComponent{
					discord.NewActionRow(discord.NewPremiumButton(c.Bot.billing.PlusSKU())),
				}
			}
			return c.Event.CreateMessage(msg)
		},
	}
}
