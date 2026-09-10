-- name: CreateMoodCheckIn :one
INSERT INTO "MoodCheckIn" ("userId", "mood", "intensity", "activity", "note", "nextCheckIn")
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: RecentMoodCheckIns :many
SELECT * FROM "MoodCheckIn"
WHERE "userId" = $1
ORDER BY "createdAt" DESC
LIMIT $2;

-- name: LastMoodCheckIn :one
SELECT * FROM "MoodCheckIn"
WHERE "userId" = $1
ORDER BY "createdAt" DESC
LIMIT 1;

-- name: MoodCheckInsSince :many
SELECT * FROM "MoodCheckIn"
WHERE "userId" = $1 AND "createdAt" >= $2
ORDER BY "createdAt" ASC;

-- name: CountMoodCheckIns :one
SELECT COUNT(*) FROM "MoodCheckIn";
