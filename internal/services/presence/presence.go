package presence

import (
	"context"
	"log/slog"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/gateway"
)

var activities = []string{
	"with your wellbeing in mind",
	"/checkin for a mood check",
	"here whenever you need to talk",
	"/coping for a calm moment",
}

func Run(ctx context.Context, client *bot.Client) {
	select {
	case <-ctx.Done():
		return
	case <-time.After(10 * time.Second):
	}

	i := 0
	apply := func() {
		if client.ShardManager == nil {
			return
		}
		activity := activities[i%len(activities)]
		i++
		for gw := range client.ShardManager.Shards() {
			if gw.Status() != gateway.StatusReady {
				continue
			}
			if err := client.SetPresenceForShard(ctx, gw.ShardID(),
				gateway.WithListeningActivity(activity),
				gateway.WithOnlineStatus(discord.OnlineStatusOnline),
			); err != nil {
				slog.Warn("set presence failed", slog.Int("shard", gw.ShardID()), slog.String("err", err.Error()))
			}
		}
	}
	apply()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			apply()
		}
	}
}
