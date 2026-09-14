package discord

import (
	"context"
	"sort"
	"time"

	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgo/sharding"
)

type ShardStatus struct {
	ID             int    `json:"id"`
	State          string `json:"state"`
	LatencyMs      int64  `json:"latencyMs"`
	Guilds         int    `json:"guilds"`
	Users          int    `json:"users"`
	LastReady      string `json:"lastReady,omitempty"`
	LastDisconnect string `json:"lastDisconnect,omitempty"`
	Resumes        int    `json:"resumes"`
	Disconnects    int    `json:"disconnects"`
}

type StatusReport struct {
	Status        string        `json:"status"`
	Version       string        `json:"version"`
	StartedAt     string        `json:"startedAt"`
	UptimeSeconds int64         `json:"uptimeSeconds"`
	ShardCount    int           `json:"shardCount"`
	Guilds        int           `json:"guilds"`
	Users         int           `json:"users"`
	Shards        []ShardStatus `json:"shards"`
	GeneratedAt   string        `json:"generatedAt"`
}

func (b *Bot) GuildCount() int {
	if b.client == nil {
		return 0
	}
	return b.client.Caches.GuildsLen()
}

func (b *Bot) UserCount() int {
	if b.client == nil {
		return 0
	}
	total := 0
	for g := range b.client.Caches.Guilds() {
		total += g.MemberCount
	}
	return total
}

func (b *Bot) readShardMeta(id int) (state string, latencyMs int64, lastReady, lastDisconnect string, resumes, disconnects int) {
	b.shardMu.Lock()
	defer b.shardMu.Unlock()
	m := b.shardMeta[id]
	if m == nil {
		return "unknown", 0, "", "", 0, 0
	}
	state = m.state
	if state == "" {
		state = "unknown"
	}
	latencyMs = m.latencyMs
	if !m.lastReady.IsZero() {
		lastReady = m.lastReady.UTC().Format(time.RFC3339)
	}
	if !m.lastDisconnect.IsZero() {
		lastDisconnect = m.lastDisconnect.UTC().Format(time.RFC3339)
	}
	resumes = m.resumes
	disconnects = m.disconnects
	return
}

func (b *Bot) pollShardLatency() {
	if b.client == nil || b.client.ShardManager == nil {
		return
	}
	for gw := range b.client.ShardManager.Shards() {
		id := gw.ShardID()
		if _, already := b.latencyPolling.LoadOrStore(id, true); already {
			continue
		}
		go func(gw gateway.Gateway, id int) {
			defer b.latencyPolling.Delete(id)
			latency := gw.Latency().Milliseconds()
			status := gw.Status()
			b.markShard(id, func(m *shardInfo) {
				m.latencyMs = latency
				if status == gateway.StatusReady {
					m.state = "ready"
				} else {
					m.state = "connecting"
				}
			})
		}(gw, id)
	}
}

func (b *Bot) RunShardLatencyPoll(ctx context.Context) {
	b.pollShardLatency()
	t := time.NewTicker(15 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			b.pollShardLatency()
		}
	}
}

func (b *Bot) StatusReport() StatusReport {
	if cached := b.statusCache.Load(); cached != nil {
		return *cached
	}
	return b.computeStatusReport()
}

func (b *Bot) RunStatusRefresh(ctx context.Context) {
	refresh := func() {
		rep := b.computeStatusReport()
		b.statusCache.Store(&rep)
	}
	refresh()

	t := time.NewTicker(10 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			refresh()
		}
	}
}

func (b *Bot) computeStatusReport() StatusReport {
	now := time.Now().UTC()
	rep := StatusReport{
		Status:        "ok",
		Version:       Version(),
		StartedAt:     b.startAt.UTC().Format(time.RFC3339),
		UptimeSeconds: int64(time.Since(b.startAt).Seconds()),
		GeneratedAt:   now.Format(time.RFC3339),
	}
	if b.client == nil {
		rep.Status = "starting"
		return rep
	}

	rep.Guilds = b.client.Caches.GuildsLen()

	sm := b.client.ShardManager
	if sm == nil {
		rep.ShardCount = 1
		rep.Users = b.UserCount()
		return rep
	}

	rep.ShardCount = b.ShardCount()

	perGuilds := map[int]int{}
	perUsers := map[int]int{}
	for g := range b.client.Caches.Guilds() {
		sid := sharding.ShardIDByGuild(g.ID, rep.ShardCount)
		perGuilds[sid]++
		perUsers[sid] += g.MemberCount
		rep.Users += g.MemberCount
	}

	degraded := false
	for gw := range sm.Shards() {
		id := gw.ShardID()
		s := ShardStatus{
			ID:     id,
			Guilds: perGuilds[id],
			Users:  perUsers[id],
		}
		s.State, s.LatencyMs, s.LastReady, s.LastDisconnect, s.Resumes, s.Disconnects = b.readShardMeta(id)
		if s.State != "ready" {
			degraded = true
		}
		rep.Shards = append(rep.Shards, s)
	}
	sort.Slice(rep.Shards, func(i, j int) bool { return rep.Shards[i].ID < rep.Shards[j].ID })
	if degraded {
		rep.Status = "degraded"
	}
	return rep
}
