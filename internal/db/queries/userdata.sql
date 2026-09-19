-- name: GetSafetyPlan :one
SELECT * FROM "SafetyPlan" WHERE "userId" = $1;

-- name: UpsertSafetyPlan :exec
INSERT INTO "SafetyPlan" ("userId", "data", "updatedAt")
VALUES ($1, $2, CURRENT_TIMESTAMP)
ON CONFLICT ("userId") DO UPDATE SET "data" = EXCLUDED."data", "updatedAt" = CURRENT_TIMESTAMP;

-- name: DeleteSafetyPlan :exec
DELETE FROM "SafetyPlan" WHERE "userId" = $1;

-- name: EraseMoodCheckIns :exec
DELETE FROM "MoodCheckIn" WHERE "userId" = $1;

-- name: EraseGhostLetters :exec
DELETE FROM "GhostLetter" WHERE "userId" = $1;

-- name: EraseCopingToolUsage :exec
DELETE FROM "CopingToolUsage" WHERE "userId" = $1;

-- name: EraseCrisisEvents :exec
DELETE FROM "CrisisEvent" WHERE "userId" = $1;

-- name: EraseJournalEntries :exec
DELETE FROM "JournalEntry" WHERE "userId" = $1;

-- name: EraseGratitudeEntries :exec
DELETE FROM "GratitudeEntry" WHERE "userId" = $1;

-- name: EraseCopingPlans :exec
DELETE FROM "CopingPlan" WHERE "userId" = $1;

-- name: EraseFavoriteCopingTools :exec
DELETE FROM "FavoriteCopingTool" WHERE "userId" = $1;

-- name: EraseUserPreferences :exec
DELETE FROM "UserPreferences" WHERE "id" = $1;

-- name: EraseWebSessions :exec
DELETE FROM "WebSession" WHERE "userId" = $1;

-- name: DetachFeedback :exec
UPDATE "Feedback" SET "userId" = NULL WHERE "userId" = $1;

-- name: DetachReports :exec
UPDATE "Report" SET "userId" = NULL WHERE "userId" = $1;

-- name: DetachSystemLogs :exec
UPDATE "SystemLog" SET "userId" = NULL WHERE "userId" = $1;

-- name: ScrubUser :exec
UPDATE "User" SET "username" = 'Deleted user' WHERE "id" = $1;
