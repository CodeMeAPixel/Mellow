-- name: CreateSystemLog :one
INSERT INTO "SystemLog" (
    "guildId", "userId", "logType", "title", "description", "metadata", "severity"
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: RecentSystemLogs :many
SELECT * FROM "SystemLog"
WHERE (sqlc.narg('log_type')::text IS NULL OR "logType" = sqlc.narg('log_type'))
ORDER BY "createdAt" DESC
LIMIT $1;
