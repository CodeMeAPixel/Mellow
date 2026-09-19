ALTER TABLE "UserPreferences"
    DROP COLUMN IF EXISTS "weeklyRecap",
    DROP COLUMN IF EXISTS "dailyPrompt",
    DROP COLUMN IF EXISTS "quietStart",
    DROP COLUMN IF EXISTS "quietEnd",
    DROP COLUMN IF EXISTS "lastRecapAt",
    DROP COLUMN IF EXISTS "lastPromptAt",
    DROP COLUMN IF EXISTS "customPersona";
