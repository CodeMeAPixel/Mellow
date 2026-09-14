package discord

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/CodeMeAPixel/Mellow/internal/ai"
	"github.com/CodeMeAPixel/Mellow/internal/billing"
	"github.com/CodeMeAPixel/Mellow/internal/config"
	"github.com/CodeMeAPixel/Mellow/internal/db"
	"github.com/CodeMeAPixel/Mellow/internal/github"
	"github.com/CodeMeAPixel/Mellow/internal/omniplex"
	"github.com/CodeMeAPixel/Mellow/internal/services/syslog"
	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgo/sharding"
	"github.com/disgoorg/snowflake/v2"
)

const contextMenuCheckMessage = "Check on this message"

type Bot struct {
	cfg      *config.Config
	store    *db.Store
	ai       *ai.Client
	syslog   *syslog.Logger
	omni     *omniplex.Client
	gh       *github.Client
	billing  *billing.Service
	client   *bot.Client
	commands map[string]*Command
	cooldns  map[string]time.Time
	startAt  time.Time

	gamesMu sync.Mutex
	games   map[string]*wordGame

	readyShards sync.Map

	shardMu        sync.Mutex
	shardMeta      map[int]*shardInfo
	latencyPolling sync.Map

	statusCache atomic.Pointer[StatusReport]
}

type shardInfo struct {
	lastReady      time.Time
	lastDisconnect time.Time
	resumes        int
	disconnects    int
	state          string
	latencyMs      int64
}

func New(cfg *config.Config, store *db.Store, aiClient *ai.Client, sl *syslog.Logger) (*Bot, error) {
	b := &Bot{
		cfg:       cfg,
		store:     store,
		ai:        aiClient,
		syslog:    sl,
		omni:      omniplex.New(cfg.OmniplexBaseURL, cfg.OmniplexToken, cfg.ClientID),
		gh:        github.New(cfg.GitHubRepo, cfg.GitHubToken),
		billing:   billing.New(store, cfg.SKUMellowPlus),
		commands:  map[string]*Command{},
		cooldns:   map[string]time.Time{},
		games:     map[string]*wordGame{},
		shardMeta: map[int]*shardInfo{},
		startAt:   time.Now(),
	}
	for _, c := range b.buildRegistry() {
		b.commands[c.Name] = c
	}

	setBrand(brandInfo{
		Website: cfg.WebsiteURL,
		Docs:    cfg.DocsURL,
		Support: cfg.SupportURL,
		Invite:  cfg.InviteURL,
		Source:  cfg.SourceURL,
	})

	intents := gateway.WithIntents(
		gateway.IntentGuilds,
		gateway.IntentGuildMessages,
		gateway.IntentDirectMessages,
	)

	client, err := disgo.New(cfg.Token,
		bot.WithShardManagerConfigOpts(
			sharding.WithAutoScaling(true),
			sharding.WithGatewayConfigOpts(intents),
			sharding.WithCloseHandler(b.onShardClose),
		),
		bot.WithCacheConfigOpts(cache.WithCaches(cache.FlagGuilds|cache.FlagChannels|cache.FlagRoles)),
		bot.WithEventListenerFunc(b.onReady),
		bot.WithEventListenerFunc(b.onResumed),
		bot.WithEventListenerFunc(b.onGuildsReady),
		bot.WithEventListenerFunc(b.onInteraction),
		bot.WithEventListenerFunc(b.onComponent),
		bot.WithEventListenerFunc(b.onModal),
		bot.WithEventListenerFunc(b.onMessage),
		bot.WithEventListenerFunc(b.onGuildReady),
		bot.WithEventListenerFunc(b.onGuildJoin),
		bot.WithEventListenerFunc(b.onGuildLeave),
		bot.WithEventListenerFunc(b.onEntitlementCreate),
		bot.WithEventListenerFunc(b.onEntitlementUpdate),
		bot.WithEventListenerFunc(b.onEntitlementDelete),
	)
	if err != nil {
		return nil, err
	}
	b.client = client
	return b, nil
}

func (b *Bot) Client() *bot.Client  { return b.client }
func (b *Bot) StartedAt() time.Time { return b.startAt }

func (b *Bot) SendLog(ctx context.Context, channelID int64, content string) error {
	_, err := b.client.Rest.CreateMessage(snowflake.ID(channelID), discord.MessageCreate{
		Content:         content,
		AllowedMentions: &discord.AllowedMentions{},
	})
	return err
}

func (b *Bot) Open(ctx context.Context) error { return b.client.OpenShardManager(ctx) }
func (b *Bot) Close(ctx context.Context)      { b.client.Close(ctx) }

func (b *Bot) ShardCount() int {
	if b.client.ShardManager == nil {
		return 1
	}
	n := 0
	for range b.client.ShardManager.Shards() {
		n++
	}
	if n == 0 {
		return 1
	}
	return n
}

func (b *Bot) Deploy(ctx context.Context) error {
	appID := b.client.ApplicationID
	if appID == 0 {
		if id, err := snowflake.Parse(b.cfg.ClientID); err == nil {
			appID = id
		} else {
			return fmt.Errorf("cannot resolve application id: %w", err)
		}
	}

	var global, private []discord.ApplicationCommandCreate
	for _, c := range b.commands {
		if c.Private {
			private = append(private, c.create())
		} else {
			global = append(global, c.create())
		}
	}
	global = append(global, discord.MessageCommandCreate{Name: contextMenuCheckMessage})

	force := os.Getenv("MELLOW_FORCE_DEPLOY") != ""
	current, _ := b.client.Rest.GetGlobalCommands(appID, false)
	if stale := staleNames(current, global); len(stale) > 0 {
		slog.Info("wiping stale global commands", slog.Any("commands", stale))
	}
	if !force && sameCommandSet(current, global) {
		slog.Info("global commands already in sync", slog.Int("count", len(global)))
	} else {
		if _, err := b.client.Rest.SetGlobalCommands(appID, global); err != nil {
			return fmt.Errorf("set global commands: %w", err)
		}
		slog.Info("global commands synced (bulk overwrite)", slog.Int("now", len(global)), slog.Int("was", len(current)))
	}

	if b.cfg.PrivateGuildID != "" {
		gid, err := snowflake.Parse(b.cfg.PrivateGuildID)
		if err != nil {
			return fmt.Errorf("parse PRIVATE_GUILD_ID: %w", err)
		}
		guildCurrent, _ := b.client.Rest.GetGuildCommands(appID, gid, false)
		if stale := staleNames(guildCurrent, private); len(stale) > 0 {
			slog.Info("wiping stale private guild commands", slog.Any("commands", stale))
		}
		if _, err := b.client.Rest.SetGuildCommands(appID, gid, private); err != nil {
			return fmt.Errorf("set guild commands: %w", err)
		}
		slog.Info("private guild commands synced (bulk overwrite)", slog.String("guild", b.cfg.PrivateGuildID), slog.Int("now", len(private)), slog.Int("was", len(guildCurrent)))
	}
	return nil
}

func staleNames(current []discord.ApplicationCommand, desired []discord.ApplicationCommandCreate) []string {
	want := make(map[string]struct{}, len(desired))
	for _, d := range desired {
		want[fmt.Sprintf("%d\x00%s", d.Type(), d.CommandName())] = struct{}{}
	}
	var out []string
	for _, c := range current {
		if _, ok := want[fmt.Sprintf("%d\x00%s", c.Type(), c.Name())]; !ok {
			out = append(out, c.Name())
		}
	}
	return out
}

func commandKey(name, desc string, typ discord.ApplicationCommandType) string {
	return fmt.Sprintf("%d\x00%s\x00%s", typ, name, desc)
}

func sameCommandSet(current []discord.ApplicationCommand, desired []discord.ApplicationCommandCreate) bool {
	if len(current) != len(desired) {
		return false
	}
	have := make(map[string]struct{}, len(current))
	for _, c := range current {
		desc := ""
		if sc, ok := c.(discord.SlashCommand); ok {
			desc = sc.Description
		}
		have[commandKey(c.Name(), desc, c.Type())] = struct{}{}
	}
	for _, d := range desired {
		desc := ""
		if sc, ok := d.(discord.SlashCommandCreate); ok {
			desc = sc.Description
		}
		if _, ok := have[commandKey(d.CommandName(), desc, d.Type())]; !ok {
			return false
		}
	}
	return true
}

func (b *Bot) onReady(e *events.Ready) {
	shardID := e.ShardID()
	slog.Info("gateway ready", slog.String("user", b.client.ID().String()), slog.Int("shard", shardID))
	if self, ok := b.client.Caches.SelfUser(); ok {
		br := brand()
		br.AvatarURL = self.EffectiveAvatarURL()
		setBrand(br)
	}

	b.markShard(shardID, func(m *shardInfo) {
		m.lastReady = time.Now()
		m.state = "ready"
	})

	ctx := context.Background()
	if _, seen := b.readyShards.LoadOrStore(shardID, true); seen {
		b.syslog.Shard(ctx, shardID, "reconnected", "A fresh session was established after a disconnect.", "warning")
	} else {
		b.syslog.Shard(ctx, shardID, "connected", "Session established.", "info")
	}
}

func (b *Bot) onResumed(e *events.Resumed) {
	slog.Info("gateway resumed", slog.Int("shard", e.ShardID()))
	b.markShard(e.ShardID(), func(m *shardInfo) {
		m.resumes++
		m.state = "ready"
	})
	b.syslog.Shard(context.Background(), e.ShardID(), "resumed", "The existing session was resumed after a brief drop.", "info")
}

func (b *Bot) markShard(id int, fn func(*shardInfo)) {
	b.shardMu.Lock()
	defer b.shardMu.Unlock()
	m := b.shardMeta[id]
	if m == nil {
		m = &shardInfo{}
		b.shardMeta[id] = m
	}
	fn(m)
}

func (b *Bot) onGuildsReady(e *events.GuildsReady) {
	n := b.client.Caches.GuildsLen()
	slog.Info("guilds loaded", slog.Int("shard", e.ShardID()), slog.Int("guilds", n))
	b.syslog.Shard(context.Background(), e.ShardID(), "guilds loaded", fmt.Sprintf("%d guild(s) available.", n), "info")
}

func (b *Bot) onShardClose(gw gateway.Gateway, err error, reconnect bool) {
	msg := "connection closed"
	if err != nil {
		msg = err.Error()
	}
	sev := "error"
	state := "disconnected"
	if reconnect {
		sev = "warning"
		state = "reconnecting"
	}
	b.markShard(gw.ShardID(), func(m *shardInfo) {
		m.disconnects++
		m.lastDisconnect = time.Now()
		m.state = state
	})
	slog.Warn("shard closed", slog.Int("shard", gw.ShardID()), slog.String("err", msg), slog.Bool("reconnect", reconnect))
	b.syslog.Shard(context.Background(), gw.ShardID(), "disconnected", msg, sev)
}

func (b *Bot) onInteraction(e *events.ApplicationCommandInteractionCreate) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	user := e.User()
	userID := int64(user.ID)
	if _, err := b.store.UpsertUser(ctx, userID, user.Username); err != nil {
		slog.Warn("user upsert failed", slog.String("err", err.Error()))
	}

	var guildID *int64
	if gid := e.GuildID(); gid != nil {
		v := int64(*gid)
		guildID = &v
		if g, ok := e.Guild(); ok {
			if _, err := b.store.UpsertGuild(ctx, int64(g.ID), g.Name, int64(g.OwnerID)); err != nil {
				slog.Warn("guild upsert failed", slog.String("err", err.Error()))
			}
		}
	}

	switch e.Data.Type() {
	case discord.ApplicationCommandTypeMessage:
		b.handleCheckMessage(ctx, e, userID, guildID)
		return
	case discord.ApplicationCommandTypeSlash:
	default:
		return
	}

	data := e.SlashCommandInteractionData()
	cmd, ok := b.commands[data.CommandName()]
	if !ok {
		return
	}

	if cmd.GuildOnly && guildID == nil {
		_ = e.CreateMessage(discord.MessageCreate{
			Content: "This command can only be used in a server.",
			Flags:   discord.MessageFlagEphemeral,
		})
		return
	}

	if cmd.OwnerOnly && !b.isOwner(ctx, userID) {
		_ = e.CreateMessage(discord.MessageCreate{
			Content: "This command is restricted to Mellow owners.",
			Flags:   discord.MessageFlagEphemeral,
		})
		return
	}

	if len(cmd.RequiredPerms) > 0 {
		if m := e.Member(); m == nil || !hasAnyPerm(m.Permissions, cmd.RequiredPerms) {
			_ = e.CreateMessage(discord.MessageCreate{
				Content: "You lack the Discord permissions required for this command.",
				Flags:   discord.MessageFlagEphemeral,
			})
			return
		}
	}

	if len(cmd.RequiredRoles) > 0 {
		allowed, msg := b.checkRole(ctx, userID, cmd.RequiredRoles)
		if !allowed {
			_ = e.CreateMessage(discord.MessageCreate{Content: msg, Flags: discord.MessageFlagEphemeral})
			return
		}
	}

	cooldown := cmd.Cooldown
	if cmd.PremiumCooldown > 0 && b.billing.HasPlusFrom(e.Entitlements()) {
		cooldown = cmd.PremiumCooldown
	}
	if cooldown > 0 {
		key := cmd.Name + ":" + strconv.FormatInt(userID, 10)
		if until, active := b.cooldns[key]; active && time.Now().Before(until) {
			wait := time.Until(until).Round(time.Second)
			_ = e.CreateMessage(discord.MessageCreate{
				Content: fmt.Sprintf("Slow down please, try again in %s.", wait),
				Flags:   discord.MessageFlagEphemeral,
			})
			return
		}
		b.cooldns[key] = time.Now().Add(cooldown)
	}

	c := &Ctx{Event: e, Data: data, Store: b.store, AI: b.ai, Bot: b, UserID: userID, GuildID: guildID}
	err := cmd.Run(ctx, c)
	b.syslog.Command(ctx, userID, guildID, cmd.Name, cmd.Category, err == nil)
	if err != nil {
		slog.Error("command failed", slog.String("cmd", cmd.Name), slog.String("err", err.Error()))
		_ = c.ReplyEphemeral("Something went wrong running that command.")
	}
}

func (b *Bot) isOwner(ctx context.Context, userID int64) bool {
	idStr := strconv.FormatInt(userID, 10)
	for _, o := range b.cfg.OwnerIDs {
		if o == idStr {
			return true
		}
	}
	if owners, err := b.store.MellowOwners(ctx); err == nil {
		for _, o := range owners {
			if o == idStr {
				return true
			}
		}
	}
	if u, err := b.store.GetUser(ctx, userID); err == nil {
		switch db.UserRole(u) {
		case "OWNER", "ADMIN":
			return true
		}
	}
	return false
}

func hasAnyPerm(have discord.Permissions, want []discord.Permissions) bool {
	if have.Has(discord.PermissionAdministrator) {
		return true
	}
	for _, p := range want {
		if have.Has(p) {
			return true
		}
	}
	return false
}

func (b *Bot) checkRole(ctx context.Context, userID int64, required []string) (bool, string) {
	u, err := b.store.GetUser(ctx, userID)
	if err != nil {
		return false, "You need to be registered in the system to use this command."
	}
	if u.IsBanned {
		return false, "You do not have access to this command."
	}
	role := db.UserRole(u)
	if role == "OWNER" {
		return true, ""
	}
	for _, r := range required {
		if strings.EqualFold(r, role) {
			return true, ""
		}
	}
	return false, "You lack the required role for this command."
}

func (b *Bot) SyncGuilds(ctx context.Context) {
	type guildRef struct {
		id      int64
		name    string
		ownerID int64
	}
	var guilds []guildRef
	for g := range b.client.Caches.Guilds() {
		guilds = append(guilds, guildRef{int64(g.ID), g.Name, int64(g.OwnerID)})
	}

	synced := 0
	for _, g := range guilds {
		if _, err := b.store.UpsertGuild(ctx, g.id, g.name, g.ownerID); err == nil {
			synced++
		}
	}
	slog.Info("guild sync", slog.Int("synced", synced))
}

func (b *Bot) RunGuildSync(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	case <-time.After(20 * time.Second):
	}
	b.SyncGuilds(ctx)

	t := time.NewTicker(30 * time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			b.SyncGuilds(ctx)
		}
	}
}

func (b *Bot) onGuildReady(e *events.GuildReady) {
	g := e.Guild
	if _, err := b.store.UpsertGuild(context.Background(), int64(g.ID), g.Name, int64(g.OwnerID)); err != nil {
		slog.Warn("guild upsert failed", slog.String("err", err.Error()))
	}
}

func (b *Bot) onGuildJoin(e *events.GuildJoin) {
	ctx := context.Background()
	g := e.Guild
	if _, err := b.store.UpsertGuild(ctx, int64(g.ID), g.Name, int64(g.OwnerID)); err != nil {
		slog.Warn("guild upsert failed", slog.String("err", err.Error()))
	}
	b.syslog.GuildMembership(ctx, true, int64(g.ID), g.Name, b.client.Caches.GuildsLen())
}

func (b *Bot) onGuildLeave(e *events.GuildLeave) {
	ctx := context.Background()
	name := e.Guild.Name
	if err := b.store.DeleteGuild(ctx, int64(e.Guild.ID)); err != nil && !errors.Is(err, db.ErrNotFound) {
		slog.Warn("guild delete failed", slog.String("err", err.Error()))
	}
	b.syslog.GuildMembership(ctx, false, int64(e.Guild.ID), name, b.client.Caches.GuildsLen())
}
