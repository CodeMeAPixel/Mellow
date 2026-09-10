package syslog

import (
	"context"
	"log/slog"
	"strconv"
	"strings"

	"github.com/CodeMeAPixel/Mellow/internal/db"
)

type Sender interface {
	SendLog(ctx context.Context, channelID int64, content string) error
}

type Logger struct {
	store        *db.Store
	sender       Sender
	supportGuild int64
	opsChannel   int64
}

func New(store *db.Store, supportGuildID, opsChannelID string) *Logger {
	l := &Logger{store: store}
	if n, err := strconv.ParseInt(supportGuildID, 10, 64); err == nil {
		l.supportGuild = n
	}
	if n, err := strconv.ParseInt(opsChannelID, 10, 64); err == nil {
		l.opsChannel = n
	}
	return l
}

func (l *Logger) SetSender(s Sender) { l.sender = s }

func (l *Logger) write(ctx context.Context, in db.NewSystemLog) {
	if l == nil || l.store == nil {
		return
	}
	if err := l.store.CreateSystemLog(ctx, in); err != nil {
		slog.Warn("syslog write failed", slog.String("err", err.Error()), slog.String("type", in.LogType))
	}
}

func glyph(severity string) string {
	switch severity {
	case "warning":
		return "⚠️ "
	case "error":
		return "⛔ "
	case "critical":
		return "\U0001f6a8 "
	default:
		return ""
	}
}

func line(title, desc, severity string) string {
	var b strings.Builder
	b.WriteString(glyph(severity))
	b.WriteString("**")
	b.WriteString(title)
	b.WriteString("**")
	if strings.TrimSpace(desc) != "" {
		b.WriteString(" — ")
		b.WriteString(strings.ReplaceAll(desc, "\n", " "))
	}
	return b.String()
}

func (l *Logger) fanout(ctx context.Context, guildID *int64, title, desc, severity string) {
	if l.sender == nil {
		return
	}
	content := line(title, desc, severity)
	sent := map[int64]bool{}
	sendTo := func(chID int64) {
		if chID == 0 || sent[chID] {
			return
		}
		sent[chID] = true
		if err := l.sender.SendLog(ctx, chID, content); err != nil {
			slog.Warn("syslog channel post failed", slog.Int64("channel", chID), slog.String("err", err.Error()))
		}
	}
	sendTo(l.opsChannel)

	seenGuilds := map[int64]bool{}
	postGuild := func(gid int64) {
		if gid == 0 || seenGuilds[gid] {
			return
		}
		seenGuilds[gid] = true
		g, err := l.store.GetGuild(ctx, gid)
		if err != nil || g.SystemChannelId == nil || *g.SystemChannelId == "" {
			return
		}
		if chID, err := strconv.ParseInt(*g.SystemChannelId, 10, 64); err == nil {
			sendTo(chID)
		}
	}
	if guildID != nil {
		postGuild(*guildID)
	}
	postGuild(l.supportGuild)
}

func strptr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (l *Logger) Command(ctx context.Context, userID int64, guildID *int64, name, category string, ok bool) {
	sev := "info"
	title := "Command used: " + name
	if !ok {
		sev = "error"
		title = "Command failed: " + name
	}
	l.write(ctx, db.NewSystemLog{
		GuildID: guildID, UserID: &userID, LogType: "command",
		Title: title, Description: strptr("category: " + category), Severity: sev,
	})
}

func (l *Logger) Crisis(ctx context.Context, userID int64, guildID *int64, level, summary string) {
	title := "Crisis signal detected (" + level + ")"
	desc := summary
	if desc == "" {
		desc = "A message from <@" + strconv.FormatInt(userID, 10) + "> was flagged."
	}
	l.write(ctx, db.NewSystemLog{
		GuildID: guildID, UserID: &userID, LogType: "crisis",
		Title: title, Description: strptr(summary), Severity: "warning",
	})
	l.fanout(ctx, guildID, title, desc, "critical")
}

func (l *Logger) Event(ctx context.Context, logType, title, description, severity string) {
	l.write(ctx, db.NewSystemLog{
		LogType: logType, Title: title, Description: strptr(description), Severity: severity,
	})
	l.fanout(ctx, nil, title, description, severity)
}

func (l *Logger) Shard(ctx context.Context, shardID int, title, detail, severity string) {
	l.Event(ctx, "gateway", "Shard "+strconv.Itoa(shardID)+": "+title, detail, severity)
}

func (l *Logger) GuildMembership(ctx context.Context, joined bool, guildID int64, name string, total int) {
	action := "Left"
	sev := "warning"
	if joined {
		action, sev = "Joined", "info"
	}
	desc := name + " (`" + strconv.FormatInt(guildID, 10) + "`), now in " + strconv.Itoa(total) + " servers"
	l.Event(ctx, "guild", action+" a server", desc, sev)
}

func (l *Logger) GuildEvent(ctx context.Context, guildID int64, logType, title, description, severity string) {
	gid := guildID
	l.write(ctx, db.NewSystemLog{
		GuildID: &gid, LogType: logType, Title: title, Description: strptr(description), Severity: severity,
	})
	l.fanout(ctx, &gid, title, description, severity)
}
