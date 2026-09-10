package discord

import (
	"sort"
	"strings"
	"time"

	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgo/sharding"
)

type ShardStatus struct {
	ID        int    `json:"id"`
	State     string `json:"state"`
	LatencyMs int64  `json:"latencyMs"`
	Guilds    int    `json:"guilds"`
	Users     int    `json:"users"`
	LastReady string `json:"lastReady,omitempty"`
	Resumes   int    `json:"resumes"`
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

func (b *Bot) StatusReport() StatusReport {
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
			ID:        id,
			State:     strings.ToLower(gw.Status().String()),
			LatencyMs: gw.Latency().Milliseconds(),
			Guilds:    perGuilds[id],
			Users:     perUsers[id],
		}
		b.shardMu.Lock()
		if m := b.shardMeta[id]; m != nil {
			if !m.lastReady.IsZero() {
				s.LastReady = m.lastReady.UTC().Format(time.RFC3339)
			}
			s.Resumes = m.resumes
		}
		b.shardMu.Unlock()
		if gw.Status() != gateway.StatusReady {
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
