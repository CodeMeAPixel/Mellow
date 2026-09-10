package db_test

import (
	"context"
	"os"
	"testing"

	"github.com/CodeMeAPixel/Mellow/internal/crypto"
	"github.com/CodeMeAPixel/Mellow/internal/db"
)

func TestIntegrationStore(t *testing.T) {
	dsn := os.Getenv("MELLOW_IT_DSN")
	if dsn == "" {
		t.Skip("set MELLOW_IT_DSN to run the store integration test")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pool.Close() })

	enc := crypto.New(os.Getenv("MELLOW_IT_KEY"), []string{firstSalt(os.Getenv("MELLOW_IT_SALTS"))})
	st := db.NewStore(pool, enc)

	var uid int64 = 999999999000000001
	t.Cleanup(func() {
		for _, q := range []string{
			`delete from "MoodCheckIn" where "userId"=$1`,
			`delete from "GhostLetter" where "userId"=$1`,
			`delete from "Feedback" where "userId"=$1`,
			`delete from "Report" where "userId"=$1`,
			`delete from "CopingToolUsage" where "userId"=$1`,
			`delete from "UserPreferences" where id=$1`,
			`delete from "User" where id=$1`,
		} {
			pool.Exec(ctx, q, uid)
		}
	})

	if _, err := st.UpsertUser(ctx, uid, "itest"); err != nil {
		t.Fatal(err)
	}

	note := "feeling raw and honest : with colons"
	in := int32(4)
	m, err := st.CreateMoodCheckIn(ctx, db.NewMoodCheckIn{UserID: uid, Mood: "anxious", Intensity: &in, Note: &note})
	if err != nil {
		t.Fatal(err)
	}
	if m.Note == nil || *m.Note != note || m.Mood != "anxious" {
		t.Fatalf("checkin decrypt mismatch: %+v", m)
	}

	var raw string
	if err := pool.QueryRow(ctx, `select "note" from "MoodCheckIn" where id=$1`, m.ID).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	if !enc.IsEncrypted(raw) {
		t.Fatalf("stored note is not ciphertext: %q", raw)
	}

	tz := "Europe/London"
	pers := "playful"
	no := false
	p, err := st.UpdateUserPreferences(ctx, uid, db.PrefsUpdate{Timezone: &tz, AIPersonality: &pers, DisableContextLogging: &no})
	if err != nil {
		t.Fatal(err)
	}
	if p.Timezone == nil || *p.Timezone != tz || p.AiPersonality == nil || *p.AiPersonality != pers || p.DisableContextLogging {
		t.Fatalf("prefs mismatch: %+v", p)
	}

	gl, err := st.CreateGhostLetter(ctx, uid, "dear nobody, today was hard")
	if err != nil {
		t.Fatal(err)
	}
	if gl.Content != "dear nobody, today was hard" {
		t.Fatalf("ghostletter roundtrip failed: %q", gl.Content)
	}

	if err := st.CreateFeedback(ctx, uid, "great bot"); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateReport(ctx, uid, "test report"); err != nil {
		t.Fatal(err)
	}

	pc, err := st.ProfileCounts(ctx, uid)
	if err != nil {
		t.Fatal(err)
	}
	if pc.CheckIns != 1 || pc.GhostLetters != 1 {
		t.Fatalf("profile counts wrong: %+v", pc)
	}
}

func firstSalt(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			return s[:i]
		}
	}
	if s == "" {
		return "mellow-encryption-salt"
	}
	return s
}

func TestIntegrationAdmin(t *testing.T) {
	dsn := os.Getenv("MELLOW_IT_DSN")
	if dsn == "" {
		t.Skip("set MELLOW_IT_DSN to run the admin integration test")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pool.Close() })
	st := db.NewStore(pool, crypto.New("", nil))

	var uid int64 = 999999999000000002
	t.Cleanup(func() {
		pool.Exec(ctx, `delete from "UserPreferences" where id=$1`, uid)
		pool.Exec(ctx, `delete from "User" where id=$1`, uid)
		pool.Exec(ctx, `update "Mellow" set owners = array_remove(owners, $1) where id=1`, "999999999000000002")
	})

	if _, err := st.UpsertUser(ctx, uid, "admin-itest"); err != nil {
		t.Fatal(err)
	}

	if err := st.SetUserRole(ctx, uid, "MOD"); err != nil {
		t.Fatalf("SetUserRole: %v", err)
	}
	u, err := st.GetUser(ctx, uid)
	if err != nil {
		t.Fatal(err)
	}
	if got := db.UserRole(u); got != "MOD" {
		t.Fatalf("role = %q, want MOD", got)
	}

	if err := st.SetUserBan(ctx, uid, true, nil, nil); err != nil {
		t.Fatal(err)
	}
	if u, _ = st.GetUser(ctx, uid); !u.IsBanned {
		t.Fatal("user not banned")
	}
	if err := st.SetUserBan(ctx, uid, false, nil, nil); err != nil {
		t.Fatal(err)
	}

	if err := st.AddMellowOwner(ctx, "999999999000000002"); err != nil {
		t.Fatal(err)
	}
	owners, err := st.MellowOwners(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, o := range owners {
		if o == "999999999000000002" {
			found = true
		}
	}
	if !found {
		t.Fatal("owner not added")
	}
	if err := st.RemoveMellowOwner(ctx, "999999999000000002"); err != nil {
		t.Fatal(err)
	}

	if _, err := st.RawMellow(ctx); err != nil {
		t.Fatalf("RawMellow: %v", err)
	}
	if _, err := st.UpdateMellowFlags(ctx, nil, nil, nil, nil); err != nil {
		t.Fatalf("UpdateMellowFlags noop: %v", err)
	}
}

func TestIntegrationContextQueries(t *testing.T) {
	dsn := os.Getenv("MELLOW_IT_DSN")
	if dsn == "" {
		t.Skip("set MELLOW_IT_DSN")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pool.Close() })
	st := db.NewStore(pool, crypto.New(os.Getenv("MELLOW_IT_KEY"), []string{firstSalt(os.Getenv("MELLOW_IT_SALTS"))}))

	var a, b int64 = 999999999000000003, 999999999000000004
	t.Cleanup(func() {
		for _, id := range []int64{a, b} {
			pool.Exec(ctx, `delete from "ConversationHistory" where "userId"=$1`, id)
			pool.Exec(ctx, `delete from "UserPreferences" where id=$1`, id)
			pool.Exec(ctx, `delete from "User" where id=$1`, id)
		}
	})
	st.UpsertUser(ctx, a, "ctx-a")
	st.UpsertUser(ctx, b, "ctx-b")

	ch := "chan-777"
	st.AddConversationMessage(ctx, db.NewConversationMessage{UserID: a, Content: "i feel anxious about sleep", ChannelID: &ch})
	st.AddConversationMessage(ctx, db.NewConversationMessage{UserID: b, Content: "hello from another user", ChannelID: &ch})

	cc, err := st.RecentChannelContext(ctx, ch, a, 8)
	if err != nil {
		t.Fatalf("RecentChannelContext: %v", err)
	}
	if len(cc) != 1 || cc[0].Username != "ctx-b" || cc[0].Content != "hello from another user" {
		t.Fatalf("channel context wrong: %+v", cc)
	}

	sum, err := st.ConversationSummarySource(ctx, a, 7)
	if err != nil {
		t.Fatalf("ConversationSummarySource: %v", err)
	}
	if len(sum) == 0 || sum[0] != "i feel anxious about sleep" {
		t.Fatalf("summary source wrong: %+v", sum)
	}
}
