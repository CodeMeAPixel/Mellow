# Patch Notes

User-facing release notes for Mellow. The newest entry is at the top. Each `##`
heading is `## v<version> - <friendly title>`; everything under it until the next
heading is the body that gets published to the bot listing.

## v0.4.0 - Crisis alerts for your server

Crisis alerts now actually reach your server. If you set an alert channel with
`/guildsettings` (or on the dashboard), a high or critical signal in your server
now posts a short alert there, as the docs have always described.

- It names the person and links to the message. It never repeats what they wrote
  and doesn't ping anyone.
- The person still receives support resources from Mellow directly.
- One person triggers at most one alert per server every ten minutes.
- Direct messages never create alerts.
- Prefer not to receive them? Turn them off with `crisis_alerts:false`, or clear
  the alert channel.

Also in this release:

- `/docs` and `/support` now link to the right places.
- The new commands and dashboard are covered in the docs at docs.mymellow.xyz.

## v0.3.0 - The Mellow dashboard

Mellow now has a home on the web at mymellow.xyz. Sign in with Discord to manage
your preferences, see your Mellow+ status, and configure the servers you manage,
all without a slash command. Purchases still happen in Discord itself, and the
dashboard links you straight to the store page.

On the dashboard:

- Preferences, including your country, quiet hours, and conversation style.
- A mood chart built from your `/checkin` history.
- **Download my data** and **Delete my data**. Both are free, always.
- Server settings for anyone with Manage Server, including clearing a channel.

New for everyone (all free):

- **Safety plan.** Write down your warning signs, what helps, and who to reach
  out to, while you're feeling okay. `/safetyplan` and the dashboard both work.
  It's encrypted, only you can see it, and Mellow shows it to you privately when
  you use `/crisis resources`.
- **Helplines that match where you live.** Set your country with
  `/preferences set country` and crisis resources use local numbers. Without one
  you get the same general list as before.
- **`/guided`.** Live breathing, 5-4-3-2-1 grounding, and an evening wind-down
  that walk you through step by step.
- **Quiet hours.** Tell Mellow when not to message you.
- **Weekly recap and daily prompt.** Two gentle, opt-in DMs. Both are off unless
  you turn them on.
- A public commands page at mymellow.xyz/commands with search and filters,
  including the permissions each command needs.

New with Mellow+:

- Mellow remembers more of your conversation.
- Three extra conversation styles (Coach, Reflective, Minimal) and a custom style
  note in your own words. It only changes tone and can never override safety
  behavior.
- Search your journal, and filter it by `#tags` you add when you write.
- 90 days of mood history on the dashboard, with a printable report.

Safety features, coping tools, and your data controls stay free.

Fixes:

- The status page's uptime and reconnect numbers were getting stuck after a
  restart. They update again, and restarts are now counted.
- Mellow's custom status now shows reliably instead of sometimes going missing.
- Documentation moved to docs.mymellow.xyz.

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
