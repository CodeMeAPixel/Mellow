package omniplexsync

import (
	"context"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/CodeMeAPixel/Mellow/internal/omniplex"
)

type StatsFunc func(ctx context.Context) (servers, users, shards int)
type CommandsFunc func() []omniplex.Command

type Service struct {
	cl       *omniplex.Client
	stats    StatsFunc
	commands CommandsFunc
}

func New(cl *omniplex.Client, stats StatsFunc, commands CommandsFunc) *Service {
	return &Service{cl: cl, stats: stats, commands: commands}
}

func (s *Service) Run(ctx context.Context) {
	if s.cl == nil || !s.cl.Enabled() {
		slog.Info("omniplex sync disabled (no OMNIPLEX_TOKEN / CLIENT_ID)")
		return
	}
	s.postStats(ctx)
	s.syncCommands(ctx)

	statsTick := time.NewTicker(10 * time.Minute)
	cmdTick := time.NewTicker(6 * time.Hour)
	defer statsTick.Stop()
	defer cmdTick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-statsTick.C:
			s.postStats(ctx)
		case <-cmdTick.C:
			s.syncCommands(ctx)
		}
	}
}

func (s *Service) postStats(ctx context.Context) {
	servers, users, shards := s.stats(ctx)
	err := s.cl.PostStats(ctx, omniplex.Stats{
		Servers: &servers,
		Users:   &users,
		Shards:  &shards,
		Status:  "online",
	})
	if err != nil {
		slog.Warn("omniplex stats post failed", slog.String("err", err.Error()))
	}
}

func (s *Service) syncCommands(ctx context.Context) {
	desired := s.commands()
	current, err := s.cl.GetCommands(ctx)
	if err != nil {
		slog.Warn("omniplex get commands failed", slog.String("err", err.Error()))
		return
	}
	if commandsEqual(current, desired) {
		slog.Info("omniplex commands unchanged", slog.Int("count", len(desired)))
		return
	}
	if err := s.cl.PutCommands(ctx, desired); err != nil {
		slog.Warn("omniplex put commands failed", slog.String("err", err.Error()))
		return
	}
	slog.Info("omniplex commands updated", slog.Int("count", len(desired)))
}

func commandsEqual(a, b []omniplex.Command) bool {
	if len(a) != len(b) {
		return false
	}
	key := func(c omniplex.Command) string {
		return strings.Join([]string{c.Name, c.Category, c.Description, c.Usage}, "\x00")
	}
	as := make([]string, len(a))
	bs := make([]string, len(b))
	for i := range a {
		as[i] = key(a[i])
	}
	for i := range b {
		bs[i] = key(b[i])
	}
	sort.Strings(as)
	sort.Strings(bs)
	for i := range as {
		if as[i] != bs[i] {
			return false
		}
	}
	return true
}
