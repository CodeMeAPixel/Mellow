package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"regexp"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/CodeMeAPixel/Mellow/internal/ai"
	"github.com/CodeMeAPixel/Mellow/internal/config"
	"github.com/CodeMeAPixel/Mellow/internal/crypto"
	"github.com/CodeMeAPixel/Mellow/internal/db"
	"github.com/CodeMeAPixel/Mellow/internal/discord"
	"github.com/CodeMeAPixel/Mellow/internal/github"
	"github.com/CodeMeAPixel/Mellow/internal/logger"
	"github.com/CodeMeAPixel/Mellow/internal/omniplex"
	"github.com/CodeMeAPixel/Mellow/internal/server"
	"github.com/CodeMeAPixel/Mellow/internal/services/omniplexsync"
	"github.com/CodeMeAPixel/Mellow/internal/services/presence"
	"github.com/CodeMeAPixel/Mellow/internal/services/reminder"
	"github.com/CodeMeAPixel/Mellow/internal/services/statusposter"
	"github.com/CodeMeAPixel/Mellow/internal/services/syslog"
	"github.com/joho/godotenv"
)

var version = "dev"

var bareRev = regexp.MustCompile(`^[0-9a-f]{7,40}(-dirty)?$`)

func vcsRevision() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	var rev string
	var dirty bool
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	if rev == "" {
		return ""
	}
	if len(rev) > 12 {
		rev = rev[:12]
	}
	if dirty {
		rev += "-dirty"
	}
	return rev
}

func resolveVersion(ctx context.Context, cfg *config.Config) string {
	if version != "" && version != "dev" && !bareRev.MatchString(version) {
		return version
	}
	if v := os.Getenv("VERSION"); v != "" {
		return v
	}

	tctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	gh := github.New(cfg.GitHubRepo, cfg.GitHubToken)
	if rel, err := gh.LatestRelease(tctx); err == nil && rel.TagName != "" {
		return rel.TagName
	}
	if tag, err := gh.LatestTag(tctx); err == nil && tag != "" {
		return tag
	}

	if version != "" && version != "dev" {
		return version
	}
	if rev := vcsRevision(); rev != "" {
		return rev
	}
	return "dev"
}

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", slog.String("err", err.Error()))
		os.Exit(1)
	}
	logger.Setup(cfg.LogLevel)
	version = resolveVersion(context.Background(), cfg)
	discord.SetVersion(version)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := db.Migrate(cfg.DatabaseURL); err != nil {
		slog.Error("migrate", slog.String("err", err.Error()))
		os.Exit(1)
	}
	if os.Getenv("MELLOW_MIGRATE_ONLY") != "" {
		slog.Info("migrate-only mode, exiting")
		return
	}

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("db connect", slog.String("err", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	enc := crypto.New(cfg.EncryptionKey, cfg.EncryptionSalt)
	if !enc.Enabled() {
		slog.Warn("encryption disabled: ENCRYPTION_KEY not set, sensitive fields stored in plaintext")
	}
	store := db.NewStore(pool, enc)
	sl := syslog.New(store, cfg.PrivateGuildID, cfg.LogChannelID)
	aiClient := ai.New(cfg.AnthropicAPIKey, store)
	if !aiClient.Live() {
		slog.Warn("ANTHROPIC_API_KEY not set, AI features return a fallback")
	}

	b, err := discord.New(cfg, store, aiClient, sl)
	if err != nil {
		slog.Error("discord init", slog.String("err", err.Error()))
		os.Exit(1)
	}
	sl.SetSender(b)

	if err := b.Open(ctx); err != nil {
		slog.Error("gateway open", slog.String("err", err.Error()))
		os.Exit(1)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		b.Close(shutdownCtx)
	}()

	if err := b.Deploy(ctx); err != nil {
		slog.Error("command deploy", slog.String("err", err.Error()))
	}

	go reminder.New(b.Client(), store).Run(ctx)
	go presence.Run(ctx, b.Client())
	go statusposter.New(cfg.StatusAPIURL, cfg.StatusAPIKey, version, b.ShardCount, b.GuildCount, b.UserCount).Run(ctx)
	go b.SweepGames(ctx)
	go b.RunGuildSync(ctx)
	go b.CheckForUpdates(ctx)

	omni := omniplex.New(cfg.OmniplexBaseURL, cfg.OmniplexToken, cfg.ClientID)
	go omniplexsync.New(omni,
		func(c context.Context) (servers, users, shards int) {
			return b.GuildCount(), b.UserCount(), b.ShardCount()
		},
		b.OmniplexCommands,
	).Run(ctx)

	srv := server.New(cfg.Port, cfg.APIToken, store, aiClient,
		func() any { return b.StatusReport() },
		func() (int, int) { return b.GuildCount(), b.UserCount() },
	)
	go func() {
		slog.Info("http api listening", slog.Int("port", cfg.Port))
		if err := srv.Start(); err != nil {
			slog.Error("http server", slog.String("err", err.Error()))
		}
	}()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	sl.Event(ctx, "startup", "Bot started", "mellow-go "+version+" online", "info")
	slog.Info("mellow-go running, press ctrl+c to stop")
	<-ctx.Done()
	slog.Info("shutting down")
	sl.Event(context.Background(), "shutdown", "Bot shutting down", "mellow-go received a stop signal", "warning")
}
