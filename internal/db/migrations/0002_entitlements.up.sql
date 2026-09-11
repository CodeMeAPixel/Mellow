CREATE TABLE IF NOT EXISTS "Entitlement" (
    "id"             BIGINT PRIMARY KEY,
    "skuId"          BIGINT NOT NULL,
    "applicationId"  BIGINT NOT NULL,
    "userId"         BIGINT,
    "guildId"        BIGINT,
    "type"           INTEGER NOT NULL,
    "consumed"       BOOLEAN,
    "deleted"        BOOLEAN NOT NULL DEFAULT false,
    "startsAt"       TIMESTAMP(3),
    "endsAt"         TIMESTAMP(3),
    "subscriptionId" BIGINT,
    "syncedAt"       TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS "Entitlement_userId_skuId_idx" ON "Entitlement" ("userId", "skuId");
CREATE INDEX IF NOT EXISTS "Entitlement_guildId_skuId_idx" ON "Entitlement" ("guildId", "skuId");
CREATE INDEX IF NOT EXISTS "Entitlement_deleted_endsAt_idx" ON "Entitlement" ("deleted", "endsAt");
