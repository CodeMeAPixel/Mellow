-- name: AddConversationMessage :one
INSERT INTO "ConversationHistory" (
    "userId", "content", "isAiResponse",
    "channelId", "guildId", "messageId", "contextType", "parentId", "metadata"
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: RecentConversationForUser :many
SELECT * FROM "ConversationHistory"
WHERE "userId" = $1
  AND (sqlc.narg('guild_id')::text IS NULL OR "guildId" = sqlc.narg('guild_id'))
ORDER BY "timestamp" DESC
LIMIT $2;

-- name: RecentChannelConversation :many
SELECT * FROM "ConversationHistory"
WHERE "channelId" = $1 AND "userId" <> $2
ORDER BY "timestamp" DESC
LIMIT $3;

-- name: CountConversationMessages :one
SELECT COUNT(*) FROM "ConversationHistory";

-- name: RecentChannelContext :many
SELECT ch.id, ch."userId", ch.content, ch."isAiResponse", ch.timestamp,
       ch."channelId", ch."guildId", ch."messageId", ch."contextType", ch."parentId", ch.metadata,
       u."username" AS username
FROM "ConversationHistory" ch
JOIN "User" u ON u."id" = ch."userId"
WHERE ch."channelId" = $1
  AND ch."userId" <> $2
  AND ch."isAiResponse" = false
  AND ch."timestamp" >= now() - interval '1 hour'
ORDER BY ch."timestamp" DESC
LIMIT $3;

-- name: ConversationSummarySource :many
SELECT content FROM "ConversationHistory"
WHERE "userId" = $1
  AND "isAiResponse" = false
  AND "contextType" = 'conversation'
  AND "timestamp" >= now() - ($2 || ' days')::interval
ORDER BY "timestamp" DESC
LIMIT 20;
