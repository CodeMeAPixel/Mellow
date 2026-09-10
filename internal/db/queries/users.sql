-- name: GetUser :one
SELECT * FROM "User" WHERE "id" = $1;

-- name: UpsertUser :one
INSERT INTO "User" ("id", "username")
VALUES ($1, $2)
ON CONFLICT ("id") DO UPDATE SET "username" = EXCLUDED."username"
RETURNING *;

-- name: SetUserRole :exec
UPDATE "User" SET "role" = sqlc.arg('role')::text::"Role" WHERE "id" = sqlc.arg('id');

-- name: SetUserBan :exec
UPDATE "User"
SET "isBanned" = $2, "bannedUntil" = $3, "banReason" = $4
WHERE "id" = $1;

-- name: CountUsers :one
SELECT COUNT(*) FROM "User";
