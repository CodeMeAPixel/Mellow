CREATE TABLE "BotRuntime" (
    "id" INTEGER PRIMARY KEY DEFAULT 1,
    "restartCount" BIGINT NOT NULL DEFAULT 0,
    "firstStartedAt" TIMESTAMP NOT NULL DEFAULT now(),
    "lastStartedAt" TIMESTAMP NOT NULL DEFAULT now(),
    CONSTRAINT "BotRuntime_singleton" CHECK ("id" = 1)
);
