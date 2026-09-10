-- name: CreateCrisisEvent :one
INSERT INTO "CrisisEvent" ("userId", "details", "escalated")
VALUES ($1, $2, $3)
RETURNING *;

-- name: CrisisEventsForUser :many
SELECT * FROM "CrisisEvent"
WHERE "userId" = $1
ORDER BY "detectedAt" DESC
LIMIT $2;

-- name: CountCrisisEventsForUser :one
SELECT COUNT(*) FROM "CrisisEvent" WHERE "userId" = $1;

-- name: CountEscalatedCrisisEventsForUser :one
SELECT COUNT(*) FROM "CrisisEvent" WHERE "userId" = $1 AND "escalated" = true;

-- name: CountCrisisEvents :one
SELECT COUNT(*) FROM "CrisisEvent";
