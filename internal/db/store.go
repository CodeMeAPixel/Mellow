package db

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/CodeMeAPixel/Mellow/internal/crypto"
	"github.com/CodeMeAPixel/Mellow/internal/db/gen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

type Store struct {
	pool *pgxpool.Pool
	q    *gen.Queries
	enc  *crypto.Service
}

func NewStore(pool *pgxpool.Pool, enc *crypto.Service) *Store {
	return &Store{pool: pool, q: gen.New(pool), enc: enc}
}

func (s *Store) Queries() *gen.Queries { return s.q }

func norm(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func (s *Store) e(v string) string { return s.enc.Encrypt(v) }
func (s *Store) d(v string) string { return s.enc.Decrypt(v) }

func (s *Store) ePtr(v *string) *string {
	if v == nil {
		return nil
	}
	out := s.enc.Encrypt(*v)
	return &out
}

func (s *Store) dPtr(v *string) *string {
	if v == nil {
		return nil
	}
	out := s.enc.Decrypt(*v)
	return &out
}

type AIConfig struct {
	Model       string
	Prompt      string
	Temperature float64
	MaxTokens   int32
	Enabled     bool

	CheckInTools bool
	CopingTools  bool
	GhostTools   bool
	CrisisTools  bool
}

func (s *Store) GetAIConfig(ctx context.Context) (AIConfig, error) {
	m, err := s.q.GetMellow(ctx)
	if err != nil {
		return AIConfig{}, norm(err)
	}
	cfg := AIConfig{
		Temperature:  m.Temperature,
		MaxTokens:    m.MaxTokens,
		Enabled:      m.Enabled,
		CheckInTools: m.CheckInTools,
		CopingTools:  m.CopingTools,
		GhostTools:   m.GhostTools,
		CrisisTools:  m.CrisisTools,
	}
	if m.Model != nil {
		cfg.Model = *m.Model
	}
	if m.Prompt != nil {
		cfg.Prompt = *m.Prompt
	}
	if cfg.Model == "" {
		cfg.Model = "claude-haiku-4-5"
	}
	if cfg.MaxTokens <= 0 {
		cfg.MaxTokens = 300
	}
	return cfg, nil
}

func (s *Store) UpsertUser(ctx context.Context, id int64, username string) (gen.User, error) {
	u, err := s.q.UpsertUser(ctx, gen.UpsertUserParams{ID: id, Username: username})
	return u, norm(err)
}

func (s *Store) GetUser(ctx context.Context, id int64) (gen.User, error) {
	u, err := s.q.GetUser(ctx, id)
	return u, norm(err)
}

func UserRole(u gen.User) string {
	if u.Role == nil {
		return "USER"
	}
	return fmt.Sprint(u.Role)
}

func (s *Store) CountUsers(ctx context.Context) (int64, error) {
	n, err := s.q.CountUsers(ctx)
	return n, norm(err)
}

func (s *Store) UpsertGuild(ctx context.Context, id int64, name string, ownerID int64) (gen.Guild, error) {
	g, err := s.q.UpsertGuild(ctx, gen.UpsertGuildParams{ID: id, Name: name, OwnerId: ownerID})
	return g, norm(err)
}

func (s *Store) GetGuild(ctx context.Context, id int64) (gen.Guild, error) {
	g, err := s.q.GetGuild(ctx, id)
	return g, norm(err)
}

func (s *Store) DeleteGuild(ctx context.Context, id int64) error {
	return norm(s.q.DeleteGuild(ctx, id))
}

func (s *Store) CountGuilds(ctx context.Context) (int64, error) {
	n, err := s.q.CountGuilds(ctx)
	return n, norm(err)
}

func (s *Store) EnsureUserPreferences(ctx context.Context, userID int64) (gen.UserPreferences, error) {
	p, err := s.q.EnsureUserPreferences(ctx, userID)
	return p, norm(err)
}

func (s *Store) GetUserPreferences(ctx context.Context, userID int64) (gen.UserPreferences, error) {
	p, err := s.q.GetUserPreferences(ctx, userID)
	return p, norm(err)
}

func (s *Store) SetNextCheckIn(ctx context.Context, userID int64, at *time.Time) error {
	return norm(s.q.SetNextCheckIn(ctx, gen.SetNextCheckInParams{ID: userID, NextCheckIn: at}))
}

func (s *Store) SetCheckInInterval(ctx context.Context, userID int64, minutes int32) error {
	return norm(s.q.SetCheckInInterval(ctx, gen.SetCheckInIntervalParams{ID: userID, CheckInInterval: minutes}))
}

func (s *Store) MarkReminderSent(ctx context.Context, userID int64, last time.Time, next *time.Time) error {
	return norm(s.q.MarkReminderSent(ctx, gen.MarkReminderSentParams{ID: userID, LastReminder: &last, NextCheckIn: next}))
}

func (s *Store) DueForReminder(ctx context.Context) ([]gen.UserPreferences, error) {
	rows, err := s.q.DueForReminder(ctx)
	return rows, norm(err)
}

type NewMoodCheckIn struct {
	UserID      int64
	Mood        string
	Intensity   *int32
	Activity    *string
	Note        *string
	NextCheckIn *time.Time
}

func (s *Store) CreateMoodCheckIn(ctx context.Context, in NewMoodCheckIn) (gen.MoodCheckIn, error) {
	row, err := s.q.CreateMoodCheckIn(ctx, gen.CreateMoodCheckInParams{
		UserId:      in.UserID,
		Mood:        s.e(in.Mood),
		Intensity:   in.Intensity,
		Activity:    s.ePtr(in.Activity),
		Note:        s.ePtr(in.Note),
		NextCheckIn: in.NextCheckIn,
	})
	if err != nil {
		return row, norm(err)
	}
	return s.decryptMood(row), nil
}

func (s *Store) decryptMood(m gen.MoodCheckIn) gen.MoodCheckIn {
	m.Mood = s.d(m.Mood)
	m.Note = s.dPtr(m.Note)
	m.Activity = s.dPtr(m.Activity)
	return m
}

func (s *Store) RecentMoodCheckIns(ctx context.Context, userID int64, limit int32) ([]gen.MoodCheckIn, error) {
	rows, err := s.q.RecentMoodCheckIns(ctx, gen.RecentMoodCheckInsParams{UserId: userID, Limit: limit})
	if err != nil {
		return nil, norm(err)
	}
	for i := range rows {
		rows[i] = s.decryptMood(rows[i])
	}
	return rows, nil
}

func (s *Store) LastMoodCheckIn(ctx context.Context, userID int64) (gen.MoodCheckIn, error) {
	row, err := s.q.LastMoodCheckIn(ctx, userID)
	if err != nil {
		return row, norm(err)
	}
	return s.decryptMood(row), nil
}

func (s *Store) MoodCheckInsSince(ctx context.Context, userID int64, since time.Time) ([]gen.MoodCheckIn, error) {
	rows, err := s.q.MoodCheckInsSince(ctx, gen.MoodCheckInsSinceParams{UserId: userID, CreatedAt: since})
	if err != nil {
		return nil, norm(err)
	}
	for i := range rows {
		rows[i] = s.decryptMood(rows[i])
	}
	return rows, nil
}

func (s *Store) RecordCopingToolUsage(ctx context.Context, userID int64, tool string) error {
	_, err := s.q.RecordCopingToolUsage(ctx, gen.RecordCopingToolUsageParams{UserId: userID, ToolName: tool})
	return norm(err)
}

func (s *Store) CopingToolUsagesForUser(ctx context.Context, userID int64, limit int32) ([]gen.CopingToolUsage, error) {
	rows, err := s.q.CopingToolUsagesForUser(ctx, gen.CopingToolUsagesForUserParams{UserId: userID, Limit: limit})
	return rows, norm(err)
}

func (s *Store) CopingToolUsageCounts(ctx context.Context, userID int64) ([]gen.CopingToolUsageCountsForUserRow, error) {
	rows, err := s.q.CopingToolUsageCountsForUser(ctx, userID)
	return rows, norm(err)
}

func (s *Store) CreateJournalEntry(ctx context.Context, userID int64, content string, private bool) (gen.JournalEntry, error) {
	row, err := s.q.CreateJournalEntry(ctx, gen.CreateJournalEntryParams{UserId: userID, Content: s.e(content), Private: private})
	if err != nil {
		return row, norm(err)
	}
	row.Content = s.d(row.Content)
	return row, nil
}

func (s *Store) JournalEntriesForUser(ctx context.Context, userID int64, limit, offset int32) ([]gen.JournalEntry, error) {
	rows, err := s.q.JournalEntriesForUser(ctx, gen.JournalEntriesForUserParams{UserId: userID, Limit: limit, Offset: offset})
	if err != nil {
		return nil, norm(err)
	}
	for i := range rows {
		rows[i].Content = s.d(rows[i].Content)
	}
	return rows, nil
}

func (s *Store) CountJournalEntries(ctx context.Context, userID int64) (int64, error) {
	n, err := s.q.CountJournalEntries(ctx, userID)
	return n, norm(err)
}

func (s *Store) DeleteJournalEntry(ctx context.Context, id int32, userID int64) (int64, error) {
	n, err := s.q.DeleteJournalEntry(ctx, gen.DeleteJournalEntryParams{ID: id, UserId: userID})
	return n, norm(err)
}

func (s *Store) CreateGratitudeEntry(ctx context.Context, userID int64, item string) (gen.GratitudeEntry, error) {
	row, err := s.q.CreateGratitudeEntry(ctx, gen.CreateGratitudeEntryParams{UserId: userID, Item: s.e(item)})
	if err != nil {
		return row, norm(err)
	}
	row.Item = s.d(row.Item)
	return row, nil
}

func (s *Store) GratitudeEntriesForUser(ctx context.Context, userID int64, limit int32) ([]gen.GratitudeEntry, error) {
	rows, err := s.q.GratitudeEntriesForUser(ctx, gen.GratitudeEntriesForUserParams{UserId: userID, Limit: limit})
	if err != nil {
		return nil, norm(err)
	}
	for i := range rows {
		rows[i].Item = s.d(rows[i].Item)
	}
	return rows, nil
}

func (s *Store) GetCopingPlan(ctx context.Context, userID int64) (gen.CopingPlan, error) {
	row, err := s.q.GetCopingPlan(ctx, userID)
	if err != nil {
		return row, norm(err)
	}
	row.Plan = s.d(row.Plan)
	return row, nil
}

func (s *Store) CreateCopingPlan(ctx context.Context, userID int64, plan string) (gen.CopingPlan, error) {
	row, err := s.q.CreateCopingPlan(ctx, gen.CreateCopingPlanParams{UserId: userID, Plan: s.e(plan)})
	if err != nil {
		return row, norm(err)
	}
	row.Plan = s.d(row.Plan)
	return row, nil
}

func (s *Store) UpdateCopingPlan(ctx context.Context, id int32, plan string) (gen.CopingPlan, error) {
	row, err := s.q.UpdateCopingPlan(ctx, gen.UpdateCopingPlanParams{ID: id, Plan: s.e(plan)})
	if err != nil {
		return row, norm(err)
	}
	row.Plan = s.d(row.Plan)
	return row, nil
}

func (s *Store) DeleteCopingPlan(ctx context.Context, id int32, userID int64) (int64, error) {
	n, err := s.q.DeleteCopingPlan(ctx, gen.DeleteCopingPlanParams{ID: id, UserId: userID})
	return n, norm(err)
}

func (s *Store) AddFavoriteCopingTool(ctx context.Context, userID int64, tool string) error {
	_, err := s.q.AddFavoriteCopingTool(ctx, gen.AddFavoriteCopingToolParams{UserId: userID, Tool: tool})
	return norm(err)
}

func (s *Store) FavoriteCopingTools(ctx context.Context, userID int64) ([]gen.FavoriteCopingTool, error) {
	rows, err := s.q.FavoriteCopingTools(ctx, userID)
	return rows, norm(err)
}

func (s *Store) RemoveFavoriteCopingTool(ctx context.Context, userID int64, tool string) (int64, error) {
	n, err := s.q.RemoveFavoriteCopingTool(ctx, gen.RemoveFavoriteCopingToolParams{UserId: userID, Tool: tool})
	return n, norm(err)
}

func (s *Store) CreateCrisisEvent(ctx context.Context, userID int64, details *string, escalated bool) (gen.CrisisEvent, error) {
	row, err := s.q.CreateCrisisEvent(ctx, gen.CreateCrisisEventParams{UserId: userID, Details: s.ePtr(details), Escalated: escalated})
	if err != nil {
		return row, norm(err)
	}
	row.Details = s.dPtr(row.Details)
	return row, nil
}

func (s *Store) CrisisEventsForUser(ctx context.Context, userID int64, limit int32) ([]gen.CrisisEvent, error) {
	rows, err := s.q.CrisisEventsForUser(ctx, gen.CrisisEventsForUserParams{UserId: userID, Limit: limit})
	if err != nil {
		return nil, norm(err)
	}
	for i := range rows {
		rows[i].Details = s.dPtr(rows[i].Details)
	}
	return rows, nil
}

func (s *Store) CrisisEventStats(ctx context.Context, userID int64) (total, escalated int64, err error) {
	total, err = s.q.CountCrisisEventsForUser(ctx, userID)
	if err != nil {
		return 0, 0, norm(err)
	}
	escalated, err = s.q.CountEscalatedCrisisEventsForUser(ctx, userID)
	return total, escalated, norm(err)
}

type NewConversationMessage struct {
	UserID       int64
	Content      string
	IsAIResponse bool
	ChannelID    *string
	GuildID      *string
	MessageID    *string
	ContextType  string
	ParentID     *int32
	Metadata     []byte
}

func (s *Store) AddConversationMessage(ctx context.Context, in NewConversationMessage) (gen.ConversationHistory, error) {
	if in.ContextType == "" {
		in.ContextType = "conversation"
	}
	row, err := s.q.AddConversationMessage(ctx, gen.AddConversationMessageParams{
		UserId:       in.UserID,
		Content:      s.e(in.Content),
		IsAiResponse: in.IsAIResponse,
		ChannelId:    in.ChannelID,
		GuildId:      in.GuildID,
		MessageId:    in.MessageID,
		ContextType:  in.ContextType,
		ParentId:     in.ParentID,
		Metadata:     in.Metadata,
	})
	if err != nil {
		return row, norm(err)
	}
	row.Content = s.d(row.Content)
	return row, nil
}

func (s *Store) RecentConversationForUser(ctx context.Context, userID int64, guildID *string, limit int32) ([]gen.ConversationHistory, error) {
	rows, err := s.q.RecentConversationForUser(ctx, gen.RecentConversationForUserParams{UserId: userID, GuildID: guildID, Limit: limit})
	if err != nil {
		return nil, norm(err)
	}
	for i := range rows {
		rows[i].Content = s.d(rows[i].Content)
	}
	return rows, nil
}

func (s *Store) RecentChannelConversation(ctx context.Context, channelID string, excludeUserID int64, limit int32) ([]gen.ConversationHistory, error) {
	rows, err := s.q.RecentChannelConversation(ctx, gen.RecentChannelConversationParams{ChannelId: &channelID, UserId: excludeUserID, Limit: limit})
	if err != nil {
		return nil, norm(err)
	}
	for i := range rows {
		rows[i].Content = s.d(rows[i].Content)
	}
	return rows, nil
}

type Stats struct {
	Users         int64
	Guilds        int64
	Conversations int64
	CrisisEvents  int64
	MoodCheckIns  int64
}

func (s *Store) CommunityStats(ctx context.Context) (Stats, error) {
	var st Stats
	var err error
	if st.Users, err = s.q.CountUsers(ctx); err != nil {
		return st, norm(err)
	}
	if st.Guilds, err = s.q.CountGuilds(ctx); err != nil {
		return st, norm(err)
	}
	if st.Conversations, err = s.q.CountConversationMessages(ctx); err != nil {
		return st, norm(err)
	}
	if st.CrisisEvents, err = s.q.CountCrisisEvents(ctx); err != nil {
		return st, norm(err)
	}
	if st.MoodCheckIns, err = s.q.CountMoodCheckIns(ctx); err != nil {
		return st, norm(err)
	}
	return st, nil
}

type NewSystemLog struct {
	GuildID     *int64
	UserID      *int64
	LogType     string
	Title       string
	Description *string
	Metadata    *string
	Severity    string
}

func (s *Store) CreateSystemLog(ctx context.Context, in NewSystemLog) error {
	if in.Severity == "" {
		in.Severity = "info"
	}
	_, err := s.q.CreateSystemLog(ctx, gen.CreateSystemLogParams{
		GuildId:     in.GuildID,
		UserId:      in.UserID,
		LogType:     in.LogType,
		Title:       in.Title,
		Description: in.Description,
		Metadata:    in.Metadata,
		Severity:    in.Severity,
	})
	return norm(err)
}

type PrefsUpdate struct {
	AIPersonality           *string
	Timezone                *string
	Language                *string
	ReminderMethod          *string
	CheckInInterval         *int32
	RemindersEnabled        *bool
	JournalPrivacy          *bool
	DisableContextLogging   *bool
	DisableCrisisDetection  *bool
	DisableCrisisSupportDMs *bool
}

func (s *Store) UpdateUserPreferences(ctx context.Context, userID int64, in PrefsUpdate) (gen.UserPreferences, error) {
	if _, err := s.q.EnsureUserPreferences(ctx, userID); err != nil {
		return gen.UserPreferences{}, norm(err)
	}
	p, err := s.q.UpdateUserPreferences(ctx, gen.UpdateUserPreferencesParams{
		ID:                      userID,
		AiPersonality:           in.AIPersonality,
		Timezone:                in.Timezone,
		Language:                in.Language,
		ReminderMethod:          in.ReminderMethod,
		CheckInInterval:         in.CheckInInterval,
		RemindersEnabled:        in.RemindersEnabled,
		JournalPrivacy:          in.JournalPrivacy,
		DisableContextLogging:   in.DisableContextLogging,
		DisableCrisisDetection:  in.DisableCrisisDetection,
		DisableCrisisSupportDms: in.DisableCrisisSupportDMs,
	})
	return p, norm(err)
}

func (s *Store) CreateGhostLetter(ctx context.Context, userID int64, content string) (gen.GhostLetter, error) {
	row, err := s.q.CreateGhostLetter(ctx, gen.CreateGhostLetterParams{UserId: userID, Content: s.e(content)})
	if err != nil {
		return row, norm(err)
	}
	row.Content = s.d(row.Content)
	return row, nil
}

func (s *Store) GhostLettersForUser(ctx context.Context, userID int64, limit int32) ([]gen.GhostLetter, error) {
	rows, err := s.q.GhostLettersForUser(ctx, gen.GhostLettersForUserParams{UserId: userID, Limit: limit})
	if err != nil {
		return nil, norm(err)
	}
	for i := range rows {
		rows[i].Content = s.d(rows[i].Content)
	}
	return rows, nil
}

func (s *Store) CreateFeedback(ctx context.Context, userID int64, message string) error {
	uid := userID
	_, err := s.q.CreateFeedback(ctx, gen.CreateFeedbackParams{UserId: &uid, Message: s.e(message)})
	return norm(err)
}

func (s *Store) CreateReport(ctx context.Context, userID int64, message string) error {
	uid := userID
	_, err := s.q.CreateReport(ctx, gen.CreateReportParams{UserId: &uid, Message: s.e(message)})
	return norm(err)
}

type ProfileCounts struct {
	CheckIns     int64
	Journal      int64
	Gratitude    int64
	GhostLetters int64
	CrisisEvents int64
	Escalated    int64
}

func (s *Store) ProfileCounts(ctx context.Context, userID int64) (ProfileCounts, error) {
	var pc ProfileCounts
	var err error
	if pc.Journal, err = s.q.CountJournalEntries(ctx, userID); err != nil {
		return pc, norm(err)
	}
	if pc.Gratitude, err = s.q.CountGratitudeForUser(ctx, userID); err != nil {
		return pc, norm(err)
	}
	if pc.GhostLetters, err = s.q.CountGhostLettersForUser(ctx, userID); err != nil {
		return pc, norm(err)
	}
	if pc.CrisisEvents, err = s.q.CountCrisisEventsForUser(ctx, userID); err != nil {
		return pc, norm(err)
	}
	if pc.Escalated, err = s.q.CountEscalatedCrisisEventsForUser(ctx, userID); err != nil {
		return pc, norm(err)
	}
	rows, err := s.q.RecentMoodCheckIns(ctx, gen.RecentMoodCheckInsParams{UserId: userID, Limit: 1000})
	if err != nil {
		return pc, norm(err)
	}
	pc.CheckIns = int64(len(rows))
	return pc, nil
}

type GuildSettingsUpdate struct {
	SystemChannelID       *string
	ModAlertChannelID     *string
	ModLogChannelID       *string
	AuditLogChannelID     *string
	CheckInChannelID      *string
	CopingToolLogID       *string
	ModeratorRoleID       *string
	SystemRoleID          *string
	EnableCheckIns        *bool
	EnableGhostLetters    *bool
	EnableCrisisAlerts    *bool
	SystemLogsEnabled     *bool
	Language              *string
	DisableContextLogging *bool
}

func (s *Store) UpdateGuildSettings(ctx context.Context, guildID int64, in GuildSettingsUpdate) (gen.Guild, error) {
	g, err := s.q.UpdateGuildSettings(ctx, gen.UpdateGuildSettingsParams{
		ID:                    guildID,
		SystemChannelID:       in.SystemChannelID,
		ModAlertChannelID:     in.ModAlertChannelID,
		ModLogChannelID:       in.ModLogChannelID,
		AuditLogChannelID:     in.AuditLogChannelID,
		CheckInChannelID:      in.CheckInChannelID,
		CopingToolLogID:       in.CopingToolLogID,
		ModeratorRoleID:       in.ModeratorRoleID,
		SystemRoleID:          in.SystemRoleID,
		EnableCheckIns:        in.EnableCheckIns,
		EnableGhostLetters:    in.EnableGhostLetters,
		EnableCrisisAlerts:    in.EnableCrisisAlerts,
		SystemLogsEnabled:     in.SystemLogsEnabled,
		Language:              in.Language,
		DisableContextLogging: in.DisableContextLogging,
	})
	return g, norm(err)
}

func (s *Store) DeleteConversationForUser(ctx context.Context, userID int64) (int64, error) {
	n, err := s.q.DeleteConversationForUser(ctx, userID)
	return n, norm(err)
}

func (s *Store) RawMellow(ctx context.Context) (gen.Mellow, error) {
	m, err := s.q.GetMellow(ctx)
	return m, norm(err)
}

func (s *Store) SetMellowEnabled(ctx context.Context, enabled bool) error {
	return norm(s.q.SetMellowEnabled(ctx, enabled))
}

type MellowAIConfigUpdate struct {
	Model       *string
	Prompt      *string
	Temperature *float64
	MaxTokens   *int32
}

func (s *Store) UpdateMellowAIConfig(ctx context.Context, in MellowAIConfigUpdate) (gen.Mellow, error) {
	m, err := s.q.UpdateMellowAIConfig(ctx, gen.UpdateMellowAIConfigParams{
		Model:       in.Model,
		Prompt:      in.Prompt,
		Temperature: in.Temperature,
		MaxTokens:   in.MaxTokens,
	})
	return m, norm(err)
}

func (s *Store) UpdateMellowFlags(ctx context.Context, checkIn, coping, ghost, crisis *bool) (gen.Mellow, error) {
	m, err := s.q.UpdateMellowFlags(ctx, gen.UpdateMellowFlagsParams{
		CheckInTools: checkIn, CopingTools: coping, GhostTools: ghost, CrisisTools: crisis,
	})
	return m, norm(err)
}

func (s *Store) MellowOwners(ctx context.Context) ([]string, error) {
	m, err := s.q.GetMellow(ctx)
	if err != nil {
		return nil, norm(err)
	}
	return m.Owners, nil
}

func (s *Store) AddMellowOwner(ctx context.Context, id string) error {
	owners, err := s.MellowOwners(ctx)
	if err != nil {
		return err
	}
	for _, o := range owners {
		if o == id {
			return nil
		}
	}
	return norm(s.q.SetMellowOwners(ctx, append(owners, id)))
}

func (s *Store) RemoveMellowOwner(ctx context.Context, id string) error {
	owners, err := s.MellowOwners(ctx)
	if err != nil {
		return err
	}
	out := owners[:0]
	for _, o := range owners {
		if o != id {
			out = append(out, o)
		}
	}
	return norm(s.q.SetMellowOwners(ctx, out))
}

func (s *Store) SetUserRole(ctx context.Context, id int64, role string) error {
	return norm(s.q.SetUserRole(ctx, gen.SetUserRoleParams{ID: id, Role: role}))
}

func (s *Store) SetUserBan(ctx context.Context, id int64, banned bool, until *time.Time, reason *string) error {
	return norm(s.q.SetUserBan(ctx, gen.SetUserBanParams{ID: id, IsBanned: banned, BannedUntil: until, BanReason: reason}))
}

func (s *Store) ListRecentUsers(ctx context.Context, limit int32) ([]gen.User, error) {
	rows, err := s.q.ListRecentUsers(ctx, limit)
	return rows, norm(err)
}

func (s *Store) ListRecentGuilds(ctx context.Context, limit int32) ([]gen.Guild, error) {
	rows, err := s.q.ListRecentGuilds(ctx, limit)
	return rows, norm(err)
}

func (s *Store) SetGuildBan(ctx context.Context, id int64, banned bool, reason *string) error {
	return norm(s.q.SetGuildBan(ctx, gen.SetGuildBanParams{ID: id, IsBanned: banned, BanReason: reason}))
}

func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

func (s *Store) ListFeedback(ctx context.Context, limit int32, approved *bool) ([]gen.Feedback, error) {
	rows, err := s.q.ListFeedback(ctx, gen.ListFeedbackParams{Limit: limit, Approved: approved})
	if err != nil {
		return nil, norm(err)
	}
	for i := range rows {
		rows[i].Message = s.d(rows[i].Message)
	}
	return rows, nil
}

func (s *Store) GetFeedback(ctx context.Context, id int32) (gen.Feedback, error) {
	row, err := s.q.GetFeedback(ctx, id)
	if err != nil {
		return row, norm(err)
	}
	row.Message = s.d(row.Message)
	return row, nil
}

func (s *Store) SetFeedbackApproval(ctx context.Context, id int32, approved, public, featured *bool) (gen.Feedback, error) {
	row, err := s.q.SetFeedbackApproval(ctx, gen.SetFeedbackApprovalParams{ID: id, Approved: approved, Public: public, Featured: featured})
	if err != nil {
		return row, norm(err)
	}
	row.Message = s.d(row.Message)
	return row, nil
}

func (s *Store) DeleteFeedback(ctx context.Context, id int32) (int64, error) {
	n, err := s.q.DeleteFeedback(ctx, id)
	return n, norm(err)
}

func (s *Store) CreateFeedbackReply(ctx context.Context, feedbackID int32, staffID int64, message string) error {
	sid := staffID
	_, err := s.q.CreateFeedbackReply(ctx, gen.CreateFeedbackReplyParams{FeedbackId: feedbackID, StaffId: &sid, Message: s.e(message)})
	return norm(err)
}

func (s *Store) FeedbackReplies(ctx context.Context, feedbackID int32) ([]gen.FeedbackReply, error) {
	rows, err := s.q.FeedbackReplies(ctx, feedbackID)
	if err != nil {
		return nil, norm(err)
	}
	for i := range rows {
		rows[i].Message = s.d(rows[i].Message)
	}
	return rows, nil
}

func (s *Store) ListReports(ctx context.Context, limit int32, status *string) ([]gen.Report, error) {
	rows, err := s.q.ListReports(ctx, gen.ListReportsParams{Limit: limit, Status: status})
	if err != nil {
		return nil, norm(err)
	}
	for i := range rows {
		rows[i].Message = s.d(rows[i].Message)
	}
	return rows, nil
}

func (s *Store) GetReport(ctx context.Context, id int32) (gen.Report, error) {
	row, err := s.q.GetReport(ctx, id)
	if err != nil {
		return row, norm(err)
	}
	row.Message = s.d(row.Message)
	return row, nil
}

func (s *Store) SetReportStatus(ctx context.Context, id int32, status string) (gen.Report, error) {
	row, err := s.q.SetReportStatus(ctx, gen.SetReportStatusParams{ID: id, Status: status})
	if err != nil {
		return row, norm(err)
	}
	row.Message = s.d(row.Message)
	return row, nil
}

func (s *Store) CreateReportReply(ctx context.Context, reportID int32, staffID int64, message string) error {
	sid := staffID
	_, err := s.q.CreateReportReply(ctx, gen.CreateReportReplyParams{ReportId: reportID, StaffId: &sid, Message: s.e(message)})
	return norm(err)
}

func (s *Store) ReportReplies(ctx context.Context, reportID int32) ([]gen.ReportReply, error) {
	rows, err := s.q.ReportReplies(ctx, reportID)
	if err != nil {
		return nil, norm(err)
	}
	for i := range rows {
		rows[i].Message = s.d(rows[i].Message)
	}
	return rows, nil
}

type ChannelContextMsg struct {
	Username  string
	Content   string
	Timestamp time.Time
}

func (s *Store) RecentChannelContext(ctx context.Context, channelID string, excludeUserID int64, limit int32) ([]ChannelContextMsg, error) {
	rows, err := s.q.RecentChannelContext(ctx, gen.RecentChannelContextParams{ChannelId: &channelID, UserId: excludeUserID, Limit: limit})
	if err != nil {
		return nil, norm(err)
	}
	out := make([]ChannelContextMsg, 0, len(rows))
	for _, r := range rows {
		out = append(out, ChannelContextMsg{Username: r.Username, Content: s.d(r.Content), Timestamp: r.Timestamp})
	}
	return out, nil
}

func (s *Store) ConversationSummarySource(ctx context.Context, userID int64, days int) ([]string, error) {
	d := strconv.Itoa(days)
	rows, err := s.q.ConversationSummarySource(ctx, gen.ConversationSummarySourceParams{UserId: userID, Column2: &d})
	if err != nil {
		return nil, norm(err)
	}
	for i := range rows {
		rows[i] = s.d(rows[i])
	}
	return rows, nil
}

func (s *Store) RecentSystemLogs(ctx context.Context, logType *string, limit int32) ([]gen.SystemLog, error) {
	rows, err := s.q.RecentSystemLogs(ctx, gen.RecentSystemLogsParams{Limit: limit, LogType: logType})
	return rows, norm(err)
}

func (s *Store) UpsertEntitlement(ctx context.Context, e gen.UpsertEntitlementParams) error {
	return s.q.UpsertEntitlement(ctx, e)
}

func (s *Store) MarkEntitlementDeleted(ctx context.Context, id int64) error {
	return s.q.MarkEntitlementDeleted(ctx, id)
}

func (s *Store) ActiveUserEntitlement(ctx context.Context, userID, skuID int64) (gen.Entitlement, error) {
	e, err := s.q.ActiveUserEntitlement(ctx, gen.ActiveUserEntitlementParams{UserId: &userID, SkuId: skuID})
	return e, norm(err)
}

func (s *Store) ActiveGuildEntitlement(ctx context.Context, guildID, skuID int64) (gen.Entitlement, error) {
	e, err := s.q.ActiveGuildEntitlement(ctx, gen.ActiveGuildEntitlementParams{GuildId: &guildID, SkuId: skuID})
	return e, norm(err)
}
