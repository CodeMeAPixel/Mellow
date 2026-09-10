package ai

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (c *Client) contextBlock(ctx context.Context, userID int64, opts GenOpts) string {
	var parts []string

	if s := c.conversationSummary(ctx, userID); s != "" {
		parts = append(parts, "User context summary: "+s)
	}

	if !opts.IsDM && opts.ChannelID != "" {
		if cc := c.channelContext(ctx, userID, opts.ChannelID); cc != "" {
			parts = append(parts, cc)
		}
	}

	if ln := lateNightBlock(opts.Timezone); ln != "" {
		parts = append(parts, ln)
	}

	return strings.Join(parts, "\n\n")
}

func (c *Client) conversationSummary(ctx context.Context, userID int64) string {
	msgs, err := c.store.ConversationSummarySource(ctx, userID, 7)
	if err != nil || len(msgs) == 0 {
		return ""
	}
	joined := strings.ToLower(strings.Join(msgs, " "))
	var themes []string
	add := func(theme string, keys ...string) {
		for _, k := range keys {
			if strings.Contains(joined, k) {
				themes = append(themes, theme)
				return
			}
		}
	}
	add("anxiety", "anxious", "anxiety", "panic")
	add("depression", "depressed", "depression", "hopeless")
	add("stress", "stress", "overwhelmed", "burnout")
	add("sleep", "sleep", "insomnia", "tired all")
	add("loneliness", "lonely", "alone", "isolated")
	add("grief", "grief", "loss", "passed away")
	if len(themes) == 0 {
		return ""
	}
	return "recent themes: " + strings.Join(themes, ", ")
}

func (c *Client) channelContext(ctx context.Context, userID int64, channelID string) string {
	rows, err := c.store.RecentChannelContext(ctx, channelID, userID, 8)
	if err != nil || len(rows) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("Recent messages from others in this channel (for situational awareness only):")
	for i := len(rows) - 1; i >= 0; i-- {
		r := rows[i]
		msg := r.Content
		if len(msg) > 200 {
			msg = msg[:200]
		}
		fmt.Fprintf(&b, "\n- %s: %s", r.Username, msg)
	}
	return b.String()
}

func lateNightBlock(tz string) string {
	if tz == "" {
		return ""
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return ""
	}
	h := time.Now().In(loc).Hour()
	switch {
	case h >= 0 && h < 5:
		return "Late-night companion mode: it is the middle of the night for this user. Use a calmer, gentler tone. Late-night feelings can feel more intense. Offer comfort without judgement about being awake, and gently suggest wind-down or grounding techniques if it fits."
	case h >= 5 && h < 8:
		return "Early morning mode: it is very early for this user. Use warm, gentle language; they may be groggy or unable to sleep."
	case h >= 22:
		return "Evening wind-down mode: it is late evening for this user. Use calming, reflective language and be supportive of end-of-day feelings."
	default:
		return ""
	}
}
