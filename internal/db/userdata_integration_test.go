package db_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/CodeMeAPixel/Mellow/internal/crypto"
	"github.com/CodeMeAPixel/Mellow/internal/db"
)

// Exercises the SQL added for the dashboard, safety plan, data export/erase,
// quiet hours, and Server Plus. Run with MELLOW_IT_DSN pointing at a
// throwaway database; it applies all migrations first.
func TestIntegrationUserDataAndServerPlus(t *testing.T) {
	dsn := os.Getenv("MELLOW_IT_DSN")
	if dsn == "" {
		t.Skip("set MELLOW_IT_DSN to run the user data integration test")
	}
	if err := db.Migrate(dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	enc := crypto.New(os.Getenv("MELLOW_IT_KEY"), []string{firstSalt(os.Getenv("MELLOW_IT_SALTS"))})
	st := db.NewStore(pool, enc)

	var uid int64 = 999999999000000002
	var gid int64 = 999999999000000003
	cleanup := func() {
		for _, q := range []string{
			`delete from "MoodCheckIn" where "userId"=$1`,
			`delete from "CopingToolUsage" where "userId"=$1`,
			`delete from "SafetyPlan" where "userId"=$1`,
			`delete from "WebSession" where "userId"=$1`,
			`delete from "UserPreferences" where id=$1`,
			`delete from "User" where id=$1`,
		} {
			pool.Exec(ctx, q, uid)
		}
		pool.Exec(ctx, `delete from "Guild" where id=$1`, gid)
	}
	cleanup()
	t.Cleanup(cleanup)

	if _, err := st.UpsertUser(ctx, uid, "erase-me"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.UpsertGuild(ctx, gid, "itest guild", uid); err != nil {
		t.Fatal(err)
	}

	// Quiet hours set and clear.
	start, end := int32(22), int32(7)
	if err := st.SetQuietHours(ctx, uid, &start, &end); err != nil {
		t.Fatal(err)
	}
	p, err := st.GetUserPreferences(ctx, uid)
	if err != nil || p.QuietStart == nil || *p.QuietStart != 22 || p.QuietEnd == nil || *p.QuietEnd != 7 {
		t.Fatalf("quiet hours not saved: %+v %v", p, err)
	}
	if err := st.SetQuietHours(ctx, uid, nil, nil); err != nil {
		t.Fatal(err)
	}
	if p, _ = st.GetUserPreferences(ctx, uid); p.QuietStart != nil || p.QuietEnd != nil {
		t.Fatalf("quiet hours not cleared: %+v", p)
	}

	// New preference columns.
	country, persona, yes := "GB", "short and kind", true
	p, err = st.UpdateUserPreferences(ctx, uid, db.PrefsUpdate{Country: &country, CustomPersona: &persona, WeeklyRecap: &yes, DailyPrompt: &yes})
	if err != nil {
		t.Fatal(err)
	}
	if p.Country == nil || *p.Country != "GB" || p.CustomPersona == nil || !p.WeeklyRecap || !p.DailyPrompt {
		t.Fatalf("prefs mismatch: %+v", p)
	}
	engaged, err := st.EngagementUsers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range engaged {
		found = found || e.ID == uid
	}
	if !found {
		t.Error("user with recap on should be listed for engagement")
	}

	// Safety plan round trip, and it must be stored encrypted when a key is set.
	plan := db.SafetyPlan{WarningSigns: "not sleeping", PeopleToAsk: "Sam, 555-0100"}
	if err := st.SaveSafetyPlan(ctx, uid, plan); err != nil {
		t.Fatal(err)
	}
	got, err := st.GetSafetyPlan(ctx, uid)
	if err != nil || got.WarningSigns != plan.WarningSigns || got.PeopleToAsk != plan.PeopleToAsk {
		t.Fatalf("safety plan mismatch: %+v %v", got, err)
	}
	if enc.Enabled() {
		var raw string
		if err := pool.QueryRow(ctx, `select "data" from "SafetyPlan" where "userId"=$1`, uid).Scan(&raw); err != nil {
			t.Fatal(err)
		}
		if !enc.IsEncrypted(raw) {
			t.Errorf("safety plan is stored in plaintext: %q", raw)
		}
	}

	// Check-in and coping exercise tagged with a server, then counted.
	in := int32(5)
	if _, err := st.CreateMoodCheckIn(ctx, db.NewMoodCheckIn{UserID: uid, Mood: "calm", Intensity: &in, GuildID: &gid}); err != nil {
		t.Fatal(err)
	}
	if err := st.RecordCopingToolUsageIn(ctx, uid, "breathing", &gid); err != nil {
		t.Fatal(err)
	}
	weeks, err := st.GuildActivity(ctx, gid, 4)
	if err != nil || len(weeks) != 1 || weeks[0].CheckIns != 1 || weeks[0].ToolUses != 1 {
		t.Fatalf("activity mismatch: %+v %v", weeks, err)
	}

	// Server schedule, extra alert channels, and clearing a channel with "".
	tz := "Europe/London"
	if err := st.SetGuildSchedule(ctx, gid, true, nil, 9, &tz); err != nil {
		t.Fatal(err)
	}
	if err := st.SetExtraAlertChannels(ctx, gid, []string{"111", "222"}); err != nil {
		t.Fatal(err)
	}
	ch := "555"
	if _, err := st.UpdateGuildSettings(ctx, gid, db.GuildSettingsUpdate{CheckInChannelID: &ch}); err != nil {
		t.Fatal(err)
	}
	g, err := st.GetGuild(ctx, gid)
	if err != nil || !g.PromptEnabled || g.PromptDay != nil || g.PromptHour != 9 || len(g.ExtraAlertChannelIds) != 2 ||
		g.CheckInChannelId == nil || *g.CheckInChannelId != "555" {
		t.Fatalf("guild settings mismatch: %+v %v", g, err)
	}
	empty := ""
	if _, err := st.UpdateGuildSettings(ctx, gid, db.GuildSettingsUpdate{CheckInChannelID: &empty}); err != nil {
		t.Fatal(err)
	}
	if g, _ = st.GetGuild(ctx, gid); g.CheckInChannelId != nil {
		t.Errorf("an empty string should clear the channel, got %v", *g.CheckInChannelId)
	}

	// Web sessions expire and delete.
	if err := st.CreateWebSession(ctx, "itest-hash", db.WebSession{UserID: uid, Username: "erase-me", ExpiresAt: pastOrFuture(1)}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.GetWebSession(ctx, "itest-hash"); err != nil {
		t.Fatalf("live session should load: %v", err)
	}

	// Export contains the data.
	exp, err := st.ExportUserData(ctx, uid)
	if err != nil || exp.SafetyPlan == nil || len(exp.MoodCheckIns) != 1 || len(exp.ToolUsage) != 1 {
		t.Fatalf("export mismatch: %+v %v", exp, err)
	}

	// Erase removes it all but keeps a scrubbed account row.
	if err := st.EraseUserData(ctx, uid); err != nil {
		t.Fatalf("erase: %v", err)
	}
	if _, err := st.GetSafetyPlan(ctx, uid); !errors.Is(err, db.ErrNotFound) {
		t.Errorf("safety plan should be gone, got %v", err)
	}
	if rows, _ := st.RecentMoodCheckIns(ctx, uid, 10); len(rows) != 0 {
		t.Errorf("check-ins should be gone, got %d", len(rows))
	}
	if _, err := st.GetUserPreferences(ctx, uid); !errors.Is(err, db.ErrNotFound) {
		t.Errorf("preferences should be gone, got %v", err)
	}
	if _, err := st.GetWebSession(ctx, "itest-hash"); !errors.Is(err, db.ErrNotFound) {
		t.Errorf("sessions should be gone, got %v", err)
	}
	u, err := st.GetUser(ctx, uid)
	if err != nil || u.Username != "Deleted user" {
		t.Errorf("account should remain with a scrubbed name: %+v %v", u, err)
	}
}

func pastOrFuture(days int) time.Time { return time.Now().AddDate(0, 0, days) }
