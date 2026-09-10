-- name: GetUserPreferences :one
SELECT * FROM "UserPreferences" WHERE "id" = $1;

-- name: EnsureUserPreferences :one
INSERT INTO "UserPreferences" ("id")
VALUES ($1)
ON CONFLICT ("id") DO UPDATE SET "id" = "UserPreferences"."id"
RETURNING *;

-- name: SetNextCheckIn :exec
UPDATE "UserPreferences"
SET "nextCheckIn" = $2
WHERE "id" = $1;

-- name: SetCheckInInterval :exec
UPDATE "UserPreferences"
SET "checkInInterval" = $2
WHERE "id" = $1;

-- name: MarkReminderSent :exec
UPDATE "UserPreferences"
SET "lastReminder" = $2, "nextCheckIn" = $3
WHERE "id" = $1;

-- name: DueForReminder :many
SELECT * FROM "UserPreferences"
WHERE "remindersEnabled" = true
  AND "nextCheckIn" IS NOT NULL
  AND "nextCheckIn" <= now();
