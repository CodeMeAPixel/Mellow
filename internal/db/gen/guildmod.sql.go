package gen

import (
	"context"
)

const deleteConversationForUser = `-- name: DeleteConversationForUser :execrows
DELETE FROM "ConversationHistory" WHERE "userId" = $1
`

func (q *Queries) DeleteConversationForUser(ctx context.Context, userid int64) (int64, error) {
	result, err := q.db.Exec(ctx, deleteConversationForUser, userid)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

const updateGuildSettings = `-- name: UpdateGuildSettings :one
UPDATE "Guild" SET
    "systemChannelId"       = COALESCE($2, "systemChannelId"),
    "modAlertChannelId"     = COALESCE($3, "modAlertChannelId"),
    "modLogChannelId"       = COALESCE($4, "modLogChannelId"),
    "auditLogChannelId"     = COALESCE($5, "auditLogChannelId"),
    "checkInChannelId"      = COALESCE($6, "checkInChannelId"),
    "copingToolLogId"       = COALESCE($7, "copingToolLogId"),
    "moderatorRoleId"       = COALESCE($8, "moderatorRoleId"),
    "systemRoleId"          = COALESCE($9, "systemRoleId"),
    "enableCheckIns"        = COALESCE($10, "enableCheckIns"),
    "enableGhostLetters"    = COALESCE($11, "enableGhostLetters"),
    "enableCrisisAlerts"    = COALESCE($12, "enableCrisisAlerts"),
    "systemLogsEnabled"     = COALESCE($13, "systemLogsEnabled"),
    "language"              = COALESCE($14, "language"),
    "disableContextLogging" = COALESCE($15, "disableContextLogging")
WHERE "id" = $1
RETURNING id, name, "ownerId", "joinedAt", "isBanned", "bannedUntil", "banReason", "systemRoleId", "systemChannelId", "systemLogsEnabled", "auditLogChannelId", "modAlertChannelId", "modLogChannelId", "checkInChannelId", "copingToolLogId", "enableCheckIns", "enableGhostLetters", "enableCrisisAlerts", "moderatorRoleId", "autoModEnabled", "autoModLevel", language, "disableContextLogging", "discordId"
`

type UpdateGuildSettingsParams struct {
	ID                    int64   `json:"id"`
	SystemChannelID       *string `json:"system_channel_id"`
	ModAlertChannelID     *string `json:"mod_alert_channel_id"`
	ModLogChannelID       *string `json:"mod_log_channel_id"`
	AuditLogChannelID     *string `json:"audit_log_channel_id"`
	CheckInChannelID      *string `json:"check_in_channel_id"`
	CopingToolLogID       *string `json:"coping_tool_log_id"`
	ModeratorRoleID       *string `json:"moderator_role_id"`
	SystemRoleID          *string `json:"system_role_id"`
	EnableCheckIns        *bool   `json:"enable_check_ins"`
	EnableGhostLetters    *bool   `json:"enable_ghost_letters"`
	EnableCrisisAlerts    *bool   `json:"enable_crisis_alerts"`
	SystemLogsEnabled     *bool   `json:"system_logs_enabled"`
	Language              *string `json:"language"`
	DisableContextLogging *bool   `json:"disable_context_logging"`
}

func (q *Queries) UpdateGuildSettings(ctx context.Context, arg UpdateGuildSettingsParams) (Guild, error) {
	row := q.db.QueryRow(ctx, updateGuildSettings,
		arg.ID,
		arg.SystemChannelID,
		arg.ModAlertChannelID,
		arg.ModLogChannelID,
		arg.AuditLogChannelID,
		arg.CheckInChannelID,
		arg.CopingToolLogID,
		arg.ModeratorRoleID,
		arg.SystemRoleID,
		arg.EnableCheckIns,
		arg.EnableGhostLetters,
		arg.EnableCrisisAlerts,
		arg.SystemLogsEnabled,
		arg.Language,
		arg.DisableContextLogging,
	)
	var i Guild
	err := row.Scan(
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
	)
	return i, err
}
