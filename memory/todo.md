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

- [x] Guests on a session — see `memory/guests-feature.md`. Migration 008.
      Any registered member adds/removes a name in PV; it shows in the group
      board's present list as `(مهمان)`. One session only, no account.
      **Each guest costs the member who added them one session, at that member's
      own role rate.** No separate guest role — the user cancelled that idea, so
      `/report`, settlement and the invoice all stayed unchanged.

- [x] Admin sets a member's حاضر/غایب for a session from PV — see
      `memory/admin-member-override.md`. No migration. Shares `sessionDelta` with
      the member's own buttons, so it is idempotent and cannot charge differently.
      Works on closed sessions; the member is notified.

Nothing is committed yet — everything is working-tree only on `feat-v2`.

## Open problem from production (2026-08-05)
- [ ] `Forbidden: permission_denied` on every send/edit aimed at the **group**.
      Private messages work fine. See `memory/known-issues.md` #0b for the SQL to
      run and the reasoning. Prime suspect: `HandleStart` uses `allGroups[0]`,
      which is just the newest group row in the whole DB, and stale group rows are
      never deleted — so events may be aimed at a group the bot was removed from.
- [x] Made the failure visible: the admin's confirmation now names the target
      group and warns when the board did not reach it. It used to say
      "✅ سانس ایجاد شد" even when the group post had failed.
- [ ] Decide the real fix once the SQL says which chat is being targeted: either a
      group picker (see the multi-group gap in `features.md`), or delete the
      `groups` row when the bot is removed from a group.

## Not verified
- [x] Migrations 006/007/008 applied cleanly on the server (deployed 2026-08-05).
- [ ] Guest money (+1 on add, -1 on remove) has **no automated test** — it is SQL
      in a transaction and needs a live database. Verify by hand: add a guest,
      check صورتحساب went up by one session, remove it, check it went back down.
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
