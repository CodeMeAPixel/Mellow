-- name: GetMellow :one
SELECT * FROM "Mellow" WHERE "id" = 1;

-- name: SetMellowEnabled :exec
UPDATE "Mellow" SET "enabled" = $1 WHERE "id" = 1;

-- name: UpdateMellowAIConfig :one
UPDATE "Mellow" SET
    "model"       = COALESCE(sqlc.narg('model'), "model"),
    "prompt"      = COALESCE(sqlc.narg('prompt'), "prompt"),
    "temperature" = COALESCE(sqlc.narg('temperature'), "temperature"),
    "maxTokens"   = COALESCE(sqlc.narg('max_tokens'), "maxTokens")
WHERE "id" = 1
RETURNING *;
