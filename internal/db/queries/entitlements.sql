-- name: UpsertEntitlement :exec
INSERT INTO "Entitlement" ("id", "skuId", "applicationId", "userId", "guildId", "type", "consumed", "deleted", "startsAt", "endsAt", "subscriptionId")
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT ("id") DO UPDATE SET
    "skuId" = EXCLUDED."skuId",
    "applicationId" = EXCLUDED."applicationId",
    "userId" = EXCLUDED."userId",
    "guildId" = EXCLUDED."guildId",
    "type" = EXCLUDED."type",
    "consumed" = EXCLUDED."consumed",
    "deleted" = EXCLUDED."deleted",
    "startsAt" = EXCLUDED."startsAt",
    "endsAt" = EXCLUDED."endsAt",
    "subscriptionId" = EXCLUDED."subscriptionId",
    "syncedAt" = CURRENT_TIMESTAMP;

-- name: MarkEntitlementDeleted :exec
UPDATE "Entitlement" SET "deleted" = true, "syncedAt" = CURRENT_TIMESTAMP WHERE "id" = $1;

-- name: ActiveUserEntitlement :one
SELECT * FROM "Entitlement"
WHERE "userId" = $1 AND "skuId" = $2 AND "deleted" = false
  AND ("endsAt" IS NULL OR "endsAt" > CURRENT_TIMESTAMP)
ORDER BY "id" DESC LIMIT 1;

-- name: ActiveGuildEntitlement :one
SELECT * FROM "Entitlement"
WHERE "guildId" = $1 AND "skuId" = $2 AND "deleted" = false
  AND ("endsAt" IS NULL OR "endsAt" > CURRENT_TIMESTAMP)
ORDER BY "id" DESC LIMIT 1;
