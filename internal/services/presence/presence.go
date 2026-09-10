package presence

import (
	"context"
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
	i := 0
	apply := func() {
		_ = client.SetPresence(ctx,
			gateway.WithListeningActivity(activities[i%len(activities)]),
			gateway.WithOnlineStatus(discord.OnlineStatusOnline),
		)
		i++
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
