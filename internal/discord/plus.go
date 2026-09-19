package discord

import "fmt"

const plusHistoryLen = 30

func plusOnly(feature string) string {
	return fmt.Sprintf("%s is a Mellow Plus feature. See what Plus offers with `/upgrade`. Everything about safety, coping tools, and your data stays free.", feature)
}

// activityGuild returns the server to record on a check-in or coping exercise.
// Server IDs are only stored while Server Plus is configured, since nothing
// else needs them, so a deployment without it keeps no per-server activity.
func (b *Bot) activityGuild(id *int64) *int64 {
	if b.billing == nil || !b.billing.ServerEnabled() {
		return nil
	}
	return id
}
