# Patch Notes

User-facing release notes for Mellow. The newest entry is at the top. Each `##`
heading is `## v<version> - <friendly title>`; everything under it until the next
heading is the body that gets published to the bot listing.

## v0.2.0 - Mellow+

Mellow+ is here: an optional way to support Mellow and get a few nice extras.
It never changes what's available for free, and it never touches safety
features. Check-ins, coping tools, crisis support, and everything else stay
exactly as open as they've always been.

`/upgrade` shows what's included:

- A full year of history in `/insights`, instead of the last 30 days.
- Shorter cooldowns on `/coping` and the fun commands.
- A supporter flair on `/profile`.

Also in this release:

- The website's status page now shows live per-shard detail (state, latency,
  reconnect count) straight from the bot instead of a delayed heartbeat.
- `/version` correctly reports the running release instead of a bare commit
  hash.
- Startup now clearly logs which stale commands it's removing when syncing
  with Discord.

## v0.1.0 - Mellow, rebuilt in Go

Mellow has been rewritten from the ground up. What that means for you:

- Faster, lighter, and more reliable.
- Runs with no privileged intents, so it only ever sees messages that mention it
  or that you send in DMs.
- Your journal entries, mood notes, ghost letters and conversations are encrypted
  at rest.
- Crisis support is more careful: clear signs of danger get a fixed, verified set
  of resources rather than an improvised reply, and hotline numbers are never
  AI-generated.
- New privacy controls: `/context view` and `/context clear` let you see and
  erase what Mellow remembers.
