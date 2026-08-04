# Known issues found while reading the code (2026-08-04)

Nothing here has been changed yet — this is a list of observations, not a changelog.

## 1. Broken format string in the "fully settled" message — FIXED on `feat-v2` (2026-08-04)

Was at `internal/handlers/handlers.go:224-229`:

```go
text = fmt.Sprintf(
    "✅ تسویه حساب انجام شد.\n\n"+
        "کاربر: %s\n"+
        "جلسات تسویه شده: %d"+
        ug.Name, sessions,          // <-- ug.Name concatenated INTO the format string
)
```

`ug.Name` was glued onto the format string instead of being passed as an argument,
so `%s` ate `sessions` and `%d` got nothing. Actual output was:

```
کاربر: %!s(int=3)
جلسات تسویه شده: %!d(MISSING)علی
```

Only fired when `sessions_owed` landed exactly on 0. **`go vet` does not catch
this** — the format string is not a constant, so the printf checker skips it. Keep
that in mind: a clean `go vet` is not proof that Sprintf calls in this repo are
correct. Fix was to close the format string with `",` and pass `ug.Name, sessions`
as real arguments.

## 2. `/report` always reports "has debts"

[internal/handlers/handlers_admin.go:360-361](../internal/handlers/handlers_admin.go#L360-L361) —
`hasDebts = true` is set for every member in the loop, not only for members with a
non-zero balance. The "no debts in this group" branch is therefore unreachable
whenever the group has at least one member.

## 0. A session can be charged twice — accepted by the user

Events and `/attendance` both add to `sessions_owed` and know nothing about each
other. Tapping ✅ میام **and** being listed in `/attendance` for the same session
charges the person twice, silently. The user was told this and chose to keep both
paths. Only fixable by hand via تسویه حساب. See `events-feature.md`.

## 3. Report prints raw names, unescaped, in Markdown mode — STILL OPEN

`/report` sends with `SendMessageWithMarkdown`, but the names are not escaped. A
name containing `_` or `*` will break the formatting. `escapeMarkdownV2` exists in
the same file but is **never called**.

Still true after the numeric-ID change on `feat-v2` — the field changed from
`username` to `ug.Name`, but `ug.Name` is free text the user typed at registration,
so it is if anything *more* likely to contain Markdown characters than a username was.

The fix already exists: `escapeMarkdown` in `handlers_event.go`. `/report` just
needs to call it. The event board already does.

## 4. Dead code (pre-existing — do not delete without asking)

- `escapeMarkdownV2` and `center` in `handlers_admin.go` — both unused.
- The `userColWidth` / `separatorWidth` / `sessionColWidth` consts — unused.
- `models.AttendanceRecord` — unused, and its migration is missing.
- `repository.GetAllRates` and `repository.GetUserGroups` — unused.

(`GetUserByUserName` used to be on this list; it was deleted on `feat-v2` because
the numeric-ID change orphaned it.)

## 5. README is stale — mostly fixed

The migration list, the `/attendance` argument format and the Bale/Telegram
wording were all corrected on `feat-v2`. The stale
`005_create_attendance_records.sql` entry is gone — see `migration-numbering.md`
for why that number can never come back.

## 6. `/attendance` silently skips people — STILL OPEN

A person only gets a `users` row when they press `/start` in the bot's private
chat. If an admin passes the ID of someone who never did, or an ID that is not a
member of the group, or a typo that is not even a number, that argument is
**dropped without any warning** — the admin only sees a success count and has no
idea which people were missed.

Switching from usernames to numeric IDs did not fix this; it arguably made it
easier to hit, since a mistyped digit still parses fine as a number. Worth showing
the admin an explicit "these IDs were skipped" list.

## 7. `.env` is committed to the working tree

`.env` exists in the project root with real values. It is in `.gitignore`, so it is
untracked — fine, just be careful not to paste its contents anywhere.
