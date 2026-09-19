DROP INDEX IF EXISTS "CopingToolUsage_guildId_usedAt_idx";
DROP INDEX IF EXISTS "MoodCheckIn_guildId_createdAt_idx";

ALTER TABLE "CopingToolUsage" DROP COLUMN IF EXISTS "guildId";
ALTER TABLE "MoodCheckIn"     DROP COLUMN IF EXISTS "guildId";

ALTER TABLE "Guild"
    DROP COLUMN IF EXISTS "extraAlertChannelIds",
    DROP COLUMN IF EXISTS "lastPromptAt",
    DROP COLUMN IF EXISTS "promptTimezone",
    DROP COLUMN IF EXISTS "promptHour",
    DROP COLUMN IF EXISTS "promptDay",
    DROP COLUMN IF EXISTS "promptEnabled";
