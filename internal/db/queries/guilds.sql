-- name: GetGuild :one
SELECT * FROM "Guild" WHERE "id" = $1;

-- name: UpsertGuild :one
INSERT INTO "Guild" ("id", "name", "ownerId")
VALUES ($1, $2, $3)
ON CONFLICT ("id") DO UPDATE SET "name" = EXCLUDED."name", "ownerId" = EXCLUDED."ownerId"
RETURNING *;

-- name: DeleteGuild :exec
DELETE FROM "Guild" WHERE "id" = $1;

-- name: CountGuilds :one
SELECT COUNT(*) FROM "Guild";
