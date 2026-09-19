package presence

import (
	"context"
	"log/slog"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/gateway"
)

var Activities = []string{
	"mymellow.xyz",
	"docs.mymellow.xyz",
	"keeping your wellbeing in mind",
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
		activity := Activities[i%len(Activities)]
		i++

		var ready []int
		for gw := range client.ShardManager.Shards() {
			if gw.Status() == gateway.StatusReady {
				ready = append(ready, gw.ShardID())
			}
		}
		for _, id := range ready {
			if err := client.SetPresenceForShard(ctx, id,
				gateway.WithCustomActivity(activity),
				gateway.WithOnlineStatus(discord.OnlineStatusOnline),
			); err != nil {
				slog.Warn("set presence failed", slog.Int("shard", id), slog.String("err", err.Error()))
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
