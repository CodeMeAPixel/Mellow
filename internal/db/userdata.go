package db

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/CodeMeAPixel/Mellow/internal/db/gen"
)

type SafetyPlan struct {
	WarningSigns     string    `json:"warningSigns"`
	CopingStrategies string    `json:"copingStrategies"`
	Distractions     string    `json:"distractions"`
	PeopleToAsk      string    `json:"peopleToAsk"`
	Professionals    string    `json:"professionals"`
	SafeEnvironment  string    `json:"safeEnvironment"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

func (p SafetyPlan) Empty() bool {
	return p.WarningSigns == "" && p.CopingStrategies == "" && p.Distractions == "" &&
		p.PeopleToAsk == "" && p.Professionals == "" && p.SafeEnvironment == ""
}

func (s *Store) GetSafetyPlan(ctx context.Context, userID int64) (SafetyPlan, error) {
	row, err := s.q.GetSafetyPlan(ctx, userID)
	if err != nil {
		return SafetyPlan{}, norm(err)
	}
	var p SafetyPlan
	if err := json.Unmarshal([]byte(s.d(row.Data)), &p); err != nil {
		return SafetyPlan{}, err
	}
	p.UpdatedAt = row.UpdatedAt
	return p, nil
}

func (s *Store) SaveSafetyPlan(ctx context.Context, userID int64, p SafetyPlan) error {
	if p.Empty() {
		return norm(s.q.DeleteSafetyPlan(ctx, userID))
	}
	p.UpdatedAt = time.Time{}
	raw, err := json.Marshal(p)
	if err != nil {
		return err
	}
	return norm(s.q.UpsertSafetyPlan(ctx, gen.UpsertSafetyPlanParams{UserId: userID, Data: s.e(string(raw))}))
}

func (s *Store) DeleteSafetyPlan(ctx context.Context, userID int64) error {
	return norm(s.q.DeleteSafetyPlan(ctx, userID))
}

type UserExport struct {
	ExportedAt    time.Time                 `json:"exportedAt"`
	ID            string                    `json:"id"`
	Username      string                    `json:"username"`
	JoinedAt      time.Time                 `json:"joinedAt"`
	Preferences   *gen.UserPreferences      `json:"preferences,omitempty"`
	SafetyPlan    *SafetyPlan               `json:"safetyPlan,omitempty"`
	MoodCheckIns  []gen.MoodCheckIn         `json:"moodCheckIns"`
	Journal       []gen.JournalEntry        `json:"journalEntries"`
	Gratitude     []gen.GratitudeEntry      `json:"gratitudeEntries"`
	CopingPlan    *gen.CopingPlan           `json:"copingPlan,omitempty"`
	FavoriteTools []gen.FavoriteCopingTool  `json:"favoriteCopingTools"`
	ToolUsage     []gen.CopingToolUsage     `json:"copingToolUsage"`
	GhostLetters  []gen.GhostLetter         `json:"ghostLetters"`
	CrisisEvents  []gen.CrisisEvent         `json:"crisisEvents"`
	Conversation  []gen.ConversationHistory `json:"conversationHistory"`
}

const exportLimit = 100000

// ExportUserData gathers everything Mellow stores about a user, decrypted.
func (s *Store) ExportUserData(ctx context.Context, userID int64) (UserExport, error) {
	u, err := s.GetUser(ctx, userID)
	if err != nil {
		return UserExport{}, err
	}
	out := UserExport{
		ExportedAt: time.Now().UTC(),
		ID:         strconv.FormatInt(userID, 10),
		Username:   u.Username,
		JoinedAt:   u.CreatedAt,
	}

	if p, err := s.GetUserPreferences(ctx, userID); err == nil {
		out.Preferences = &p
	} else if !errors.Is(err, ErrNotFound) {
		return out, err
	}
	if sp, err := s.GetSafetyPlan(ctx, userID); err == nil {
		out.SafetyPlan = &sp
	} else if !errors.Is(err, ErrNotFound) {
		return out, err
	}
	if cp, err := s.GetCopingPlan(ctx, userID); err == nil {
		out.CopingPlan = &cp
	} else if !errors.Is(err, ErrNotFound) {
		return out, err
	}

	if out.MoodCheckIns, err = s.RecentMoodCheckIns(ctx, userID, exportLimit); err != nil {
		return out, err
	}
	if out.Journal, err = s.JournalEntriesForUser(ctx, userID, exportLimit, 0); err != nil {
		return out, err
	}
	if out.Gratitude, err = s.GratitudeEntriesForUser(ctx, userID, exportLimit); err != nil {
		return out, err
	}
	if out.FavoriteTools, err = s.FavoriteCopingTools(ctx, userID); err != nil {
		return out, err
	}
	if out.ToolUsage, err = s.CopingToolUsagesForUser(ctx, userID, exportLimit); err != nil {
		return out, err
	}
	if out.GhostLetters, err = s.GhostLettersForUser(ctx, userID, exportLimit); err != nil {
		return out, err
	}
	if out.CrisisEvents, err = s.CrisisEventsForUser(ctx, userID, exportLimit); err != nil {
		return out, err
	}
	if out.Conversation, err = s.RecentConversationForUser(ctx, userID, nil, exportLimit); err != nil {
		return out, err
	}
	return out, nil
}

// EraseUserData deletes a user's personal data in one transaction. The User
// row itself is kept (with the name scrubbed) so bans and role history still
// apply if the same account returns; feedback, reports, and system logs are
// detached from the account instead of deleted.
func (s *Store) EraseUserData(ctx context.Context, userID int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q := s.q.WithTx(tx)

	steps := []func() error{
		func() error { _, err := q.DeleteConversationForUser(ctx, userID); return err },
		func() error { return q.EraseMoodCheckIns(ctx, userID) },
		func() error { return q.EraseGhostLetters(ctx, userID) },
		func() error { return q.EraseCopingToolUsage(ctx, userID) },
		func() error { return q.EraseCrisisEvents(ctx, userID) },
		func() error { return q.EraseJournalEntries(ctx, userID) },
		func() error { return q.EraseGratitudeEntries(ctx, userID) },
		func() error { return q.EraseCopingPlans(ctx, userID) },
		func() error { return q.EraseFavoriteCopingTools(ctx, userID) },
		func() error { return q.DeleteSafetyPlan(ctx, userID) },
		func() error { return q.EraseUserPreferences(ctx, userID) },
		func() error { return q.EraseWebSessions(ctx, userID) },
		func() error { return q.DetachFeedback(ctx, &userID) },
		func() error { return q.DetachReports(ctx, &userID) },
		func() error { return q.DetachSystemLogs(ctx, &userID) },
		func() error { return q.ScrubUser(ctx, userID) },
	}
	for _, step := range steps {
		if err := step(); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
