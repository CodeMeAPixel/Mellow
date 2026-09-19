ALTER TABLE "Guild"
    ADD COLUMN IF NOT EXISTS "promptEnabled"        BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS "promptDay"            INTEGER,
    ADD COLUMN IF NOT EXISTS "promptHour"           INTEGER NOT NULL DEFAULT 10,
    ADD COLUMN IF NOT EXISTS "promptTimezone"       TEXT,
    ADD COLUMN IF NOT EXISTS "lastPromptAt"         TIMESTAMP(3),
    ADD COLUMN IF NOT EXISTS "extraAlertChannelIds" TEXT[] NOT NULL DEFAULT '{}';

ALTER TABLE "MoodCheckIn"     ADD COLUMN IF NOT EXISTS "guildId" BIGINT;
ALTER TABLE "CopingToolUsage" ADD COLUMN IF NOT EXISTS "guildId" BIGINT;

CREATE INDEX IF NOT EXISTS "MoodCheckIn_guildId_createdAt_idx" ON "MoodCheckIn" ("guildId", "createdAt");
CREATE INDEX IF NOT EXISTS "CopingToolUsage_guildId_usedAt_idx" ON "CopingToolUsage" ("guildId", "usedAt");
