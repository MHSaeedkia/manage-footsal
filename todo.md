# TODO

## Done
- [x] Read the whole codebase and write `memory/` (project-overview, features, known-issues) — 2026-08-04.
- [x] Branch `feat-v2` created off `main` — 2026-08-04.
- [x] Fix broken `fmt.Sprintf` in the settled-to-zero message — `memory/known-issues.md` #1.
- [x] Identify users by numeric ID instead of `@username` — see `memory/identity-is-numeric-id.md`.
- [x] Futsal events with RSVP — see `memory/events-feature.md`.
      Admin creates a session (month/date/capacity), bot posts the board in the
      group and DMs everyone two buttons. Present = +1 session, same as
      `/attendance`. Answers can change until the admin closes the event. Board is
      edited in place after every answer. Each answer sends the member a bill, and
      admins can send everyone their bill at once.

- [x] Run on Bale **or** Telegram via `PLATFORM` in `.env` — see `memory/platform-switch.md`.
      Default `bale` keeps old behaviour. Wired into `.env.example`, `docker-compose.yml`
      and README. Bad values fail at boot with a clear error.

Nothing is committed yet — everything is working-tree only on `feat-v2`.

## Not verified
- [ ] **Migrations 006/007 still need a clean run.** First attempt crash-looped
      because they were numbered 005/006 and version 5 was already burned by a
      deleted migration — see `memory/migration-numbering.md`. Renumbered to
      006/007; confirm `docker-compose up` now applies both.
- [ ] The whole event flow is untested against a live Bale bot. Only `sessionDelta`
      and the board rendering have unit tests (`internal/handlers/handlers_event_test.go`).
- [ ] `PLATFORM=telegram` has never been run against the real Telegram API. Only the
      endpoint mapping is unit tested (`internal/bot/bot_test.go`). Needs one live
      smoke test with a Telegram token **and a separate database** — see the trap in
      `memory/platform-switch.md`.

## Candidate fixes (not started, not approved)
- [ ] Fix `hasDebts` always true in `/report` — `memory/known-issues.md` #2.
- [ ] Call `escapeMarkdown` on names in `/report` — #3. The helper already exists in
      `handlers_event.go`; `/report` just does not use it.
- [ ] Tell the admin which IDs `/attendance` skipped instead of silently dropping them — #6.
- [ ] Double-charge risk between events and `/attendance` — #0. User accepted it; revisit if it bites.
- [ ] Decide the fate of `AttendanceRecord` (still unused, still no migration) — #4.
- [ ] Consider a migration to drop the leftover `attendance_records` table from
      version 5 on old databases. Nothing reads it. Would be version 008.
- [ ] Consider whether `users.username` is worth keeping at all now that nothing reads it.
- [ ] Event invites are sent one-by-one inside the update loop. Fine for a small
      group; would need a queue if membership grows a lot.
