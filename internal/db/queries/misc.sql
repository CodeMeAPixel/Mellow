-- name: UpdateUserPreferences :one
UPDATE "UserPreferences" SET
    "aiPersonality"           = COALESCE(sqlc.narg('ai_personality'), "aiPersonality"),
    "timezone"                = COALESCE(sqlc.narg('timezone'), "timezone"),
    "country"                 = COALESCE(sqlc.narg('country'), "country"),
    "weeklyRecap"            = COALESCE(sqlc.narg('weekly_recap'), "weeklyRecap"),
    "dailyPrompt"            = COALESCE(sqlc.narg('daily_prompt'), "dailyPrompt"),
    "customPersona"          = COALESCE(sqlc.narg('custom_persona'), "customPersona"),
    "language"                = COALESCE(sqlc.narg('language'), "language"),
    "reminderMethod"          = COALESCE(sqlc.narg('reminder_method'), "reminderMethod"),
    "profileTheme"            = COALESCE(sqlc.narg('profile_theme'), "profileTheme"),
    "checkInInterval"         = COALESCE(sqlc.narg('check_in_interval'), "checkInInterval"),
    "remindersEnabled"        = COALESCE(sqlc.narg('reminders_enabled'), "remindersEnabled"),
    "journalPrivacy"          = COALESCE(sqlc.narg('journal_privacy'), "journalPrivacy"),
    "disableContextLogging"   = COALESCE(sqlc.narg('disable_context_logging'), "disableContextLogging"),
    "disableCrisisDetection"  = COALESCE(sqlc.narg('disable_crisis_detection'), "disableCrisisDetection"),
    "disableCrisisSupportDMs" = COALESCE(sqlc.narg('disable_crisis_support_dms'), "disableCrisisSupportDMs")
WHERE "id" = $1
RETURNING *;

-- name: CreateGhostLetter :one
INSERT INTO "GhostLetter" ("userId", "content")
VALUES ($1, $2)
RETURNING *;

-- name: GhostLettersForUser :many
SELECT * FROM "GhostLetter"
WHERE "userId" = $1
ORDER BY "createdAt" DESC
LIMIT $2;

-- name: CountGhostLettersForUser :one
SELECT COUNT(*) FROM "GhostLetter" WHERE "userId" = $1;

-- name: CreateFeedback :one
INSERT INTO "Feedback" ("userId", "message")
VALUES ($1, $2)
RETURNING *;

-- name: CreateReport :one
INSERT INTO "Report" ("userId", "message")
VALUES ($1, $2)
RETURNING *;

-- name: CountGratitudeForUser :one
SELECT COUNT(*) FROM "GratitudeEntry" WHERE "userId" = $1;
