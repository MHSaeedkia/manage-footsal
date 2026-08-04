# Running on Bale or Telegram (`PLATFORM`)

Added on branch `feat-v2`, 2026-08-04.

## How it works

Bale and Telegram expose the **same** Bot API. Only the host differs, so one
`go-telegram-bot-api/v5` client covers both — there is no second library and no
abstraction layer. The whole switch is one endpoint string:

| `PLATFORM` | endpoint |
| --- | --- |
| `bale` (default) | `https://tapi.bale.ai/bot%s/%s` |
| `telegram` | `tgbotapi.APIEndpoint` = `https://api.telegram.org/bot%s/%s` |

`bot.APIEndpointFor(platform)` does the mapping and returns an error for anything
else. `main.go` calls it **before** connecting so a typo fails fast at boot with a
clear message instead of producing confusing HTTP errors later. The value is
lowercased and trimmed first, but is otherwise exact — `"Bale"` with a capital B is
rejected on purpose rather than silently guessed at (there is a test for this).

The default is `bale`, which keeps the behaviour the project had before the switch
existed. Someone who does not set `PLATFORM` sees no change.

`Bot.Platform` stores the choice, currently only for the startup log line.

## Trap: the database does not know about platforms

There is **no platform column anywhere**. `users.telegram_id` just holds "the id on
whatever platform is running".

Bale ids and Telegram ids are different number spaces for the same human. If you
point the same database at a different `PLATFORM`, every stored id becomes
meaningless — rows will not match the people logging in, and `/attendance` with a
Telegram id could land on a row created from a Bale id.

**Use a separate database per platform.** Do not flip `PLATFORM` on a database that
already has data.

## Where it is wired

- `internal/bot/bot.go` — the constants, `APIEndpointFor`, `New(token, platform, ...)`
- `cmd/bot/main.go` — reads `PLATFORM`, validates, passes it in
- `.env.example`, `docker-compose.yml` (`PLATFORM: ${PLATFORM:-bale}`), README

`docker-compose.yml` lists every env var explicitly, so **any new env var must be
added there too** or it never reaches the container.

Related: [[project-overview]], [[identity-is-numeric-id]]
