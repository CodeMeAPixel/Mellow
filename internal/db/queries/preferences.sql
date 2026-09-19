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

-- name: SetQuietHours :exec
UPDATE "UserPreferences" SET "quietStart" = $2, "quietEnd" = $3 WHERE "id" = $1;

-- name: MarkRecapSent :exec
UPDATE "UserPreferences" SET "lastRecapAt" = $2 WHERE "id" = $1;

-- name: MarkPromptSent :exec
UPDATE "UserPreferences" SET "lastPromptAt" = $2 WHERE "id" = $1;

-- name: EngagementUsers :many
SELECT * FROM "UserPreferences" WHERE "weeklyRecap" = true OR "dailyPrompt" = true;
