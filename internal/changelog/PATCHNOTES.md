# Patch Notes

User-facing release notes for Mellow. The newest entry is at the top. Each `##`
heading is `## v<version> - <friendly title>`; everything under it until the next
heading is the body that gets published to the bot listing.

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
