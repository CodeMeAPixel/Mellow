-- name: SetGuildSchedule :exec
UPDATE "Guild" SET
    "promptEnabled"  = $2,
    "promptDay"      = $3,
    "promptHour"     = $4,
    "promptTimezone" = $5
WHERE "id" = $1;

-- name: SetExtraAlertChannels :exec
UPDATE "Guild" SET "extraAlertChannelIds" = $2 WHERE "id" = $1;

-- name: GuildsWithPrompts :many
SELECT * FROM "Guild" WHERE "promptEnabled" = true AND "isBanned" = false;

-- name: MarkGuildPromptSent :exec
UPDATE "Guild" SET "lastPromptAt" = $2 WHERE "id" = $1;

-- name: GuildWeeklyCheckIns :many
SELECT date_trunc('week', "createdAt")::timestamp AS week, COUNT(*)::bigint AS total
FROM "MoodCheckIn"
WHERE "guildId" = $1 AND "createdAt" >= $2
GROUP BY 1 ORDER BY 1;

-- name: GuildWeeklyToolUses :many
SELECT date_trunc('week', "usedAt")::timestamp AS week, COUNT(*)::bigint AS total
FROM "CopingToolUsage"
WHERE "guildId" = $1 AND "usedAt" >= $2
GROUP BY 1 ORDER BY 1;
