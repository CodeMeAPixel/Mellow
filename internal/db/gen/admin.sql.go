package gen

import (
	"context"
)

const listRecentGuilds = `-- name: ListRecentGuilds :many
SELECT id, name, "ownerId", "joinedAt", "isBanned", "bannedUntil", "banReason", "systemRoleId", "systemChannelId", "systemLogsEnabled", "auditLogChannelId", "modAlertChannelId", "modLogChannelId", "checkInChannelId", "copingToolLogId", "enableCheckIns", "enableGhostLetters", "enableCrisisAlerts", "moderatorRoleId", "autoModEnabled", "autoModLevel", language, "disableContextLogging", "discordId" FROM "Guild" ORDER BY "joinedAt" DESC LIMIT $1
`

func (q *Queries) ListRecentGuilds(ctx context.Context, limit int32) ([]Guild, error) {
	rows, err := q.db.Query(ctx, listRecentGuilds, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Guild
	for rows.Next() {
		var i Guild
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.OwnerId,
			&i.JoinedAt,
			&i.IsBanned,
			&i.BannedUntil,
			&i.BanReason,
			&i.SystemRoleId,
			&i.SystemChannelId,
			&i.SystemLogsEnabled,
			&i.AuditLogChannelId,
			&i.ModAlertChannelId,
			&i.ModLogChannelId,
			&i.CheckInChannelId,
			&i.CopingToolLogId,
			&i.EnableCheckIns,
			&i.EnableGhostLetters,
			&i.EnableCrisisAlerts,
			&i.ModeratorRoleId,
			&i.AutoModEnabled,
			&i.AutoModLevel,
			&i.Language,
			&i.DisableContextLogging,
			&i.DiscordId,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listRecentUsers = `-- name: ListRecentUsers :many
SELECT id, username, role, "createdAt", "isBanned", "bannedUntil", "banReason", "discordId" FROM "User" ORDER BY "createdAt" DESC LIMIT $1
`

func (q *Queries) ListRecentUsers(ctx context.Context, limit int32) ([]User, error) {
	rows, err := q.db.Query(ctx, listRecentUsers, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []User
	for rows.Next() {
		var i User
		if err := rows.Scan(
			&i.ID,
			&i.Username,
			&i.Role,
			&i.CreatedAt,
			&i.IsBanned,
			&i.BannedUntil,
			&i.BanReason,
			&i.DiscordId,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const setGuildBan = `-- name: SetGuildBan :exec
UPDATE "Guild" SET "isBanned" = $2, "banReason" = $3 WHERE "id" = $1
`

type SetGuildBanParams struct {
	ID        int64   `json:"id"`
	IsBanned  bool    `json:"isBanned"`
	BanReason *string `json:"banReason"`
}

func (q *Queries) SetGuildBan(ctx context.Context, arg SetGuildBanParams) error {
	_, err := q.db.Exec(ctx, setGuildBan, arg.ID, arg.IsBanned, arg.BanReason)
	return err
}

const setMellowOwners = `-- name: SetMellowOwners :exec
UPDATE "Mellow" SET "owners" = $1 WHERE "id" = 1
`

func (q *Queries) SetMellowOwners(ctx context.Context, owners []string) error {
	_, err := q.db.Exec(ctx, setMellowOwners, owners)
	return err
}

const updateMellowFlags = `-- name: UpdateMellowFlags :one
UPDATE "Mellow" SET
    "checkInTools" = COALESCE($1, "checkInTools"),
    "copingTools"  = COALESCE($2, "copingTools"),
    "ghostTools"   = COALESCE($3, "ghostTools"),
    "crisisTools"  = COALESCE($4, "crisisTools")
WHERE "id" = 1
RETURNING id, model, prompt, temperature, "presencePenalty", "frequencyPenalty", "maxTokens", enabled, "checkInTools", "copingTools", "ghostTools", "crisisTools", owners, "feedbackLogs", "reportLogs", "serverId", "adminId", "modId", "logId"
`

type UpdateMellowFlagsParams struct {
	CheckInTools *bool `json:"check_in_tools"`
	CopingTools  *bool `json:"coping_tools"`
	GhostTools   *bool `json:"ghost_tools"`
	CrisisTools  *bool `json:"crisis_tools"`
}

func (q *Queries) UpdateMellowFlags(ctx context.Context, arg UpdateMellowFlagsParams) (Mellow, error) {
	row := q.db.QueryRow(ctx, updateMellowFlags,
		arg.CheckInTools,
		arg.CopingTools,
		arg.GhostTools,
		arg.CrisisTools,
	)
	var i Mellow
	err := row.Scan(
		&i.ID,
		&i.Model,
		&i.Prompt,
		&i.Temperature,
		&i.PresencePenalty,
		&i.FrequencyPenalty,
		&i.MaxTokens,
		&i.Enabled,
		&i.CheckInTools,
		&i.CopingTools,
		&i.GhostTools,
		&i.CrisisTools,
		&i.Owners,
		&i.FeedbackLogs,
		&i.ReportLogs,
		&i.ServerId,
		&i.AdminId,
		&i.ModId,
		&i.LogId,
	)
	return i, err
}
