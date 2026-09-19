CREATE TABLE "WebSession" (
    "tokenHash" TEXT PRIMARY KEY,
    "userId"    BIGINT NOT NULL,
    "username"  TEXT NOT NULL,
    "avatar"    TEXT,
    "guildIds"  BIGINT[] NOT NULL DEFAULT '{}',
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "expiresAt" TIMESTAMP(3) NOT NULL
);
CREATE INDEX "WebSession_userId_idx" ON "WebSession" ("userId");
CREATE INDEX "WebSession_expiresAt_idx" ON "WebSession" ("expiresAt");
