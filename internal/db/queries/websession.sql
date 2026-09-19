-- name: CreateWebSession :exec
INSERT INTO "WebSession" ("tokenHash", "userId", "username", "avatar", "guildIds", "expiresAt")
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetWebSession :one
SELECT * FROM "WebSession"
WHERE "tokenHash" = $1 AND "expiresAt" > CURRENT_TIMESTAMP;

-- name: DeleteWebSession :exec
DELETE FROM "WebSession" WHERE "tokenHash" = $1;

-- name: DeleteExpiredWebSessions :exec
DELETE FROM "WebSession" WHERE "expiresAt" <= CURRENT_TIMESTAMP;
