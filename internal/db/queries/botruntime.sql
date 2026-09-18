-- name: RecordBotStart :one
INSERT INTO "BotRuntime" ("id", "restartCount", "firstStartedAt", "lastStartedAt")
VALUES (1, 1, now(), now())
ON CONFLICT ("id") DO UPDATE SET
    "restartCount" = "BotRuntime"."restartCount" + 1,
    "lastStartedAt" = now()
RETURNING *;
