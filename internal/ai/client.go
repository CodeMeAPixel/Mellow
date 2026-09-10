package ai

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/CodeMeAPixel/Mellow/internal/db"
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const defaultHistoryLen = 10

var (
	ErrDisabled = errors.New("mellow ai is disabled")
	ErrNoKey    = errors.New("anthropic api key not set")
)

type Client struct {
	api   anthropic.Client
	store *db.Store
	live  bool

	mu    sync.RWMutex
	cfg   db.AIConfig
	cfgAt time.Time
}

func New(apiKey string, store *db.Store) *Client {
	c := &Client{store: store, live: apiKey != ""}
	if c.live {
		c.api = anthropic.NewClient(option.WithAPIKey(apiKey))
	}
	return c
}

func (c *Client) Live() bool { return c.live }

func (c *Client) config(ctx context.Context) db.AIConfig {
	c.mu.RLock()
	fresh := time.Since(c.cfgAt) < time.Minute && c.cfg.Model != ""
	cached := c.cfg
	c.mu.RUnlock()
	if fresh {
		return cached
	}
	cfg, err := c.store.GetAIConfig(ctx)
	if err != nil {
		return cached
	}
	c.mu.Lock()
	c.cfg, c.cfgAt = cfg, time.Now()
	c.mu.Unlock()
	return cfg
}

func (c *Client) Config(ctx context.Context) db.AIConfig { return c.config(ctx) }

func (c *Client) RefreshConfig(ctx context.Context) db.AIConfig {
	c.mu.Lock()
	c.cfgAt = time.Time{}
	c.mu.Unlock()
	return c.config(ctx)
}

type GenOpts struct {
	GuildID     string
	ChannelID   string
	MessageID   string
	IsDM        bool
	Personality string
	Timezone    string
	Persist     bool
	HistoryLen  int32
}

func (c *Client) Generate(ctx context.Context, userID int64, userMsg string, opts GenOpts) (string, error) {
	if !c.live {
		return "", ErrNoKey
	}
	cfg := c.config(ctx)
	if !cfg.Enabled {
		return "", ErrDisabled
	}

	stable := buildStableSystem(cfg.Prompt, opts)
	volatile := c.contextBlock(ctx, userID, opts)
	msgs := c.loadHistory(ctx, userID, opts)
	msgs = append(msgs, anthropic.NewUserMessage(anthropic.NewTextBlock(userMsg)))

	out, err := c.complete(ctx, cfg, stable, volatile, msgs, cfg.Temperature)
	if err != nil {
		return "", err
	}

	if opts.Persist {
		c.persist(ctx, userID, userMsg, out, opts)
	}
	return formatForDiscord(out), nil
}

func (c *Client) oneShot(ctx context.Context, system, user string) (string, error) {
	return c.oneShotTemp(ctx, system, user, -1)
}

func (c *Client) constrainedOneShot(ctx context.Context, system, user string) (string, error) {
	return c.oneShotTemp(ctx, system, user, 0)
}

func (c *Client) oneShotTemp(ctx context.Context, system, user string, temp float64) (string, error) {
	if !c.live {
		return "", ErrNoKey
	}
	cfg := c.config(ctx)
	if !cfg.Enabled {
		return "", ErrDisabled
	}
	if temp < 0 {
		temp = cfg.Temperature
	}
	msgs := []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(user))}
	out, err := c.complete(ctx, cfg, system, "", msgs, temp)
	if err != nil {
		return "", err
	}
	return formatForDiscord(out), nil
}

func (c *Client) complete(ctx context.Context, cfg db.AIConfig, stable, volatile string, msgs []anthropic.MessageParam, temp float64) (string, error) {
	sys := []anthropic.TextBlockParam{{
		Text:         stable,
		CacheControl: anthropic.NewCacheControlEphemeralParam(),
	}}
	if strings.TrimSpace(volatile) != "" {
		sys = append(sys, anthropic.TextBlockParam{Text: volatile})
	}
	params := anthropic.MessageNewParams{
		Model:       anthropic.Model(cfg.Model),
		MaxTokens:   int64(cfg.MaxTokens),
		System:      sys,
		Messages:    msgs,
		Temperature: anthropic.Float(temp),
	}
	resp, err := c.api.Messages.New(ctx, params)
	if err != nil {
		return "", err
	}
	var b []byte
	for _, block := range resp.Content {
		if t, ok := block.AsAny().(anthropic.TextBlock); ok {
			b = append(b, t.Text...)
		}
	}
	return string(b), nil
}

func (c *Client) loadHistory(ctx context.Context, userID int64, opts GenOpts) []anthropic.MessageParam {
	limit := opts.HistoryLen
	if limit <= 0 {
		limit = defaultHistoryLen
	}
	var guildID *string
	if opts.GuildID != "" {
		guildID = &opts.GuildID
	}
	rows, err := c.store.RecentConversationForUser(ctx, userID, guildID, limit)
	if err != nil || len(rows) == 0 {
		return nil
	}
	msgs := make([]anthropic.MessageParam, 0, len(rows))
	for i := len(rows) - 1; i >= 0; i-- {
		r := rows[i]
		if r.Content == "" {
			continue
		}
		if r.IsAiResponse {
			msgs = append(msgs, anthropic.NewAssistantMessage(anthropic.NewTextBlock(r.Content)))
		} else {
			msgs = append(msgs, anthropic.NewUserMessage(anthropic.NewTextBlock(r.Content)))
		}
	}
	return collapseRoles(msgs)
}

func (c *Client) persist(ctx context.Context, userID int64, userMsg, aiMsg string, opts GenOpts) {
	var channelID, guildID, messageID *string
	if opts.ChannelID != "" {
		channelID = &opts.ChannelID
	}
	if opts.GuildID != "" {
		guildID = &opts.GuildID
	}
	if opts.MessageID != "" {
		messageID = &opts.MessageID
	}
	_, _ = c.store.AddConversationMessage(ctx, db.NewConversationMessage{
		UserID: userID, Content: userMsg, IsAIResponse: false,
		ChannelID: channelID, GuildID: guildID, MessageID: messageID,
	})
	_, _ = c.store.AddConversationMessage(ctx, db.NewConversationMessage{
		UserID: userID, Content: aiMsg, IsAIResponse: true,
		ChannelID: channelID, GuildID: guildID,
	})
}

func collapseRoles(msgs []anthropic.MessageParam) []anthropic.MessageParam {
	if len(msgs) == 0 {
		return msgs
	}
	for len(msgs) > 0 && msgs[0].Role == anthropic.MessageParamRoleAssistant {
		msgs = msgs[1:]
	}
	return msgs
}
