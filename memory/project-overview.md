# Project Overview — futsal-bot

A **Bale** messenger bot (not Telegram) written in Go + PostgreSQL. It tracks who
attended each futsal session and how much money each person owes.

## Key fact: runs on Bale **or** Telegram

Both messengers speak the same Bot API, so one `go-telegram-bot-api/v5` client
serves both — only the host differs. Chosen with `PLATFORM` in `.env`
(`bale` is the default; see `platform-switch.md`).

So all DB columns / struct fields named `telegram_id`, `telegram_chat_id` hold the
id of **whichever platform is configured** — despite the name. They are not
Telegram-specific.

All user-facing text is Persian (Farsi).

## Layers

| Path | Role |
| --- | --- |
| `cmd/bot/main.go` | entry point: env → logger → DB → migrations → update loop |
| `internal/bot/bot.go` | Bot struct, in-memory state machine, send helpers, keyboard builders |
| `internal/handlers/handlers.go` | private-chat handlers (start, text input, callbacks) |
| `internal/handlers/handlers_admin.go` | admin callbacks + group commands |
| `internal/database/repository.go` | all SQL, plain `database/sql` (no ORM) |
| `internal/models/models.go` | structs + `UserRole` enum |
| `migrations/` | goose SQL migrations, run automatically at boot |
| `pkg/logger/` | zap wrapper, configured via `LOG_LEVEL`/`LOG_FORMAT`/`LOG_OUTPUT` |

## Data model

- `users` — one row per Bale user.
- `groups` — one row per chat the bot was added to.
- `user_groups` — membership: `role`, display `name`, `sessions_owed`. This is the
  core table; `sessions_owed` is the whole ledger.
- `rates` — price per session, per `(group, role)`.

`sessions_owed` is a signed integer:
- `> 0` → user owes money (بدهکار)
- `< 0` → user is in credit (طلبکار)
- `= 0` → settled

Money is never stored. It is always computed as `sessions_owed × rate`.

## Roles

`admin`, `student`, `adult`, `half_adult` (Postgres enum `user_role`). Each role has
its own rate per group. Admin rights come from either the `admin` role in
`user_groups`, or from matching the `DEFAULT_ADMIN_ID` env var.

## State machine

Conversation state lives in a `map[int64]*models.UserState` guarded by a mutex
([internal/bot/bot.go:13-19](../internal/bot/bot.go#L13-L19)). **It is in-memory only —
restarting the bot drops every in-progress conversation.** States used:
`awaiting_name`, `awaiting_role`, `awaiting_rate`, `awaiting_settle_sessions`.

Callback data is colon-separated: `action:arg1:arg2`, e.g. `setrate:student:12`.
