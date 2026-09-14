ALTER TABLE "SystemLog" ADD CONSTRAINT "SystemLog_guildId_fkey" FOREIGN KEY ("guildId") REFERENCES "Guild" ("id");
ALTER TABLE "SystemLog" ADD CONSTRAINT "SystemLog_userId_fkey" FOREIGN KEY ("userId") REFERENCES "User" ("id");
