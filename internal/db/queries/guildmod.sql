-- name: UpdateGuildSettings :one
UPDATE "Guild" SET
    "systemChannelId"       = NULLIF(COALESCE(sqlc.narg('system_channel_id'), "systemChannelId"), ''),
    "modAlertChannelId"     = NULLIF(COALESCE(sqlc.narg('mod_alert_channel_id'), "modAlertChannelId"), ''),
    "modLogChannelId"       = NULLIF(COALESCE(sqlc.narg('mod_log_channel_id'), "modLogChannelId"), ''),
    "auditLogChannelId"     = NULLIF(COALESCE(sqlc.narg('audit_log_channel_id'), "auditLogChannelId"), ''),
    "checkInChannelId"      = NULLIF(COALESCE(sqlc.narg('check_in_channel_id'), "checkInChannelId"), ''),
    "copingToolLogId"       = NULLIF(COALESCE(sqlc.narg('coping_tool_log_id'), "copingToolLogId"), ''),
    "moderatorRoleId"       = NULLIF(COALESCE(sqlc.narg('moderator_role_id'), "moderatorRoleId"), ''),
    "systemRoleId"          = NULLIF(COALESCE(sqlc.narg('system_role_id'), "systemRoleId"), ''),
    "enableCheckIns"        = COALESCE(sqlc.narg('enable_check_ins'), "enableCheckIns"),
    "enableGhostLetters"    = COALESCE(sqlc.narg('enable_ghost_letters'), "enableGhostLetters"),
    "enableCrisisAlerts"    = COALESCE(sqlc.narg('enable_crisis_alerts'), "enableCrisisAlerts"),
    "systemLogsEnabled"     = COALESCE(sqlc.narg('system_logs_enabled'), "systemLogsEnabled"),
    "language"              = COALESCE(sqlc.narg('language'), "language"),
    "disableContextLogging" = COALESCE(sqlc.narg('disable_context_logging'), "disableContextLogging")
WHERE "id" = $1
RETURNING *;

-- name: DeleteConversationForUser :execrows
DELETE FROM "ConversationHistory" WHERE "userId" = $1;
