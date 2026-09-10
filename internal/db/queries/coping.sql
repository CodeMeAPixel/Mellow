-- name: RecordCopingToolUsage :one
INSERT INTO "CopingToolUsage" ("userId", "toolName")
VALUES ($1, $2)
RETURNING *;

-- name: CopingToolUsagesForUser :many
SELECT * FROM "CopingToolUsage"
WHERE "userId" = $1
ORDER BY "usedAt" DESC
LIMIT $2;

-- name: CopingToolUsageCountsForUser :many
SELECT "toolName", COUNT(*) AS uses
FROM "CopingToolUsage"
WHERE "userId" = $1
GROUP BY "toolName"
ORDER BY uses DESC;

-- name: CreateJournalEntry :one
INSERT INTO "JournalEntry" ("userId", "content", "private")
VALUES ($1, $2, $3)
RETURNING *;

-- name: JournalEntriesForUser :many
SELECT * FROM "JournalEntry"
WHERE "userId" = $1
ORDER BY "createdAt" DESC
LIMIT $2 OFFSET $3;

-- name: CountJournalEntries :one
SELECT COUNT(*) FROM "JournalEntry" WHERE "userId" = $1;

-- name: DeleteJournalEntry :execrows
DELETE FROM "JournalEntry" WHERE "id" = $1 AND "userId" = $2;

-- name: CreateGratitudeEntry :one
INSERT INTO "GratitudeEntry" ("userId", "item")
VALUES ($1, $2)
RETURNING *;

-- name: GratitudeEntriesForUser :many
SELECT * FROM "GratitudeEntry"
WHERE "userId" = $1
ORDER BY "createdAt" DESC
LIMIT $2;

-- name: GetCopingPlan :one
SELECT * FROM "CopingPlan"
WHERE "userId" = $1
ORDER BY "updatedAt" DESC
LIMIT 1;

-- name: CreateCopingPlan :one
INSERT INTO "CopingPlan" ("userId", "plan")
VALUES ($1, $2)
RETURNING *;

-- name: UpdateCopingPlan :one
UPDATE "CopingPlan" SET "plan" = $2
WHERE "id" = $1
RETURNING *;

-- name: DeleteCopingPlan :execrows
DELETE FROM "CopingPlan" WHERE "id" = $1 AND "userId" = $2;

-- name: AddFavoriteCopingTool :one
INSERT INTO "FavoriteCopingTool" ("userId", "tool")
VALUES ($1, $2)
RETURNING *;

-- name: FavoriteCopingTools :many
SELECT * FROM "FavoriteCopingTool"
WHERE "userId" = $1
ORDER BY "tool" ASC;

-- name: RemoveFavoriteCopingTool :execrows
DELETE FROM "FavoriteCopingTool" WHERE "userId" = $1 AND "tool" = $2;
