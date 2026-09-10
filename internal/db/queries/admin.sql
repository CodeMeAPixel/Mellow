-- name: UpdateMellowFlags :one
UPDATE "Mellow" SET
    "checkInTools" = COALESCE(sqlc.narg('check_in_tools'), "checkInTools"),
    "copingTools"  = COALESCE(sqlc.narg('coping_tools'), "copingTools"),
    "ghostTools"   = COALESCE(sqlc.narg('ghost_tools'), "ghostTools"),
    "crisisTools"  = COALESCE(sqlc.narg('crisis_tools'), "crisisTools")
WHERE "id" = 1
RETURNING *;

-- name: SetMellowOwners :exec
UPDATE "Mellow" SET "owners" = $1 WHERE "id" = 1;

-- name: ListRecentUsers :many
SELECT * FROM "User" ORDER BY "createdAt" DESC LIMIT $1;

-- name: ListRecentGuilds :many
SELECT * FROM "Guild" ORDER BY "joinedAt" DESC LIMIT $1;

-- name: SetGuildBan :exec
UPDATE "Guild" SET "isBanned" = $2, "banReason" = $3 WHERE "id" = $1;
