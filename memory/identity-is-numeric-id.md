# Decision: users are identified by numeric ID, never by @username

Decided by the user on 2026-08-04, implemented on branch `feat-v2`.

## The rule

**A numeric Bale user ID is the only accepted way to identify a person.**
`@username` is not an identity in this system anymore. Do not reintroduce
username lookups.

Reason given by the user: the numeric ID is unique. A username is not a safe key —
a person can change it, or not have one at all.

## Two different numeric IDs — do not mix them up

This is the easiest thing to get wrong in this repo:

| Value | Where | What it is |
| --- | --- | --- |
| `users.telegram_id` / `models.User.TelegramID` | the Bale-side ID | **this** is what admins type into `/attendance` |
| `users.id` / `models.UserGroup.UserID` | internal `BIGSERIAL` PK | only for joins, never shown, never typed |

`/attendance` parses the **Bale** ID and resolves it with `GetUserByTelegramID`.
Every repository call after that (`IsUserMemberOfGroup`, `AddSessionsToUser`,
`GetUserGroup`, `SettleSessions`) takes the **internal** `users.id`. So the handler
must go ID → `*models.User` → `u.ID` before touching those. Passing a Bale ID
straight into `AddSessionsToUser` would silently update nothing.

## What changed

- `handleAttendanceCommand` now does `strconv.ParseInt` per argument and calls
  `GetUserByTelegramID`. The `@` stripping is gone.
- `repository.GetUserByUserName` was **deleted** — nothing used it after the change.
- `/report` no longer runs `SELECT username FROM users WHERE id = $1` per member.
  It uses `ug.Name` (the name the person typed at registration), which
  `GetUserGroupsByGroupID` already returns. This also removed an N+1 query.

## Consequence to remember

`users.username` is still written by `GetOrCreateUser` and still on `models.User`,
but **nothing reads it for identity anymore**. It is now only incidental data. If
it is never needed, it could be dropped later — but that needs a migration, so it
was left alone.

Names in `/report` are **not unique** — two people can register the same name. The
user accepted this: readability was preferred over uniqueness for the report,
because the report is only for an admin to look at, not to identify anyone by.

Related: [[known-issues]], [[features]]
