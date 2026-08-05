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

`/attendance` adds to `sessions_owed` and knows nothing about events. Tapping
✅ میام **and** being listed in `/attendance` for the same session charges the
person twice, silently. The user was told this and chose to keep both paths. Only
fixable by hand via تسویه حساب. See `events-feature.md`.

The member's own buttons and the admin override (`وضعیت اعضا`) are **safe** with
each other — both go through `event_responses` and `sessionDelta`, so they cannot
double-charge. `/attendance` is the odd one out. Prefer the admin override; see
`admin-member-override.md`.

## 0b. `Forbidden: permission_denied` when posting the event board — OPEN

Seen in production 2026-08-05. Every failure was a send **or** edit aimed at the
group; private messages were fine.

The bot targets `groups.telegram_chat_id` of `event.GroupID`. That group comes from
`HandleStart`, which picks **`allGroups[0]`** — and `GetAllGroups` orders by
`created_at DESC`, so it is simply the **newest group row in the whole database**,
not a group the user chose. See `#7` below.

Nothing ever deletes a `groups` row, so a group the bot was removed from stays in
the table forever and can still be the newest. That makes stale group rows the
prime suspect whenever the group side fails but private chats work.

Checks, in order:

```sql
SELECT id, telegram_chat_id, title, created_at FROM groups ORDER BY created_at DESC;
SELECT id, group_id, month, session_date, group_message_id FROM events ORDER BY id;
```

Then confirm the bot is still a member of that exact chat and may post there (on
Bale a group can be set so only admins send messages).

Note the timeline in the reported logs: event 1 had a non-zero `group_message_id`,
and `refreshEventBoard` returns early when that is 0 — so event 1 **did** post
successfully once. The bot lost group access after that, it was never missing.

The board post failure used to be **invisible to the admin** — they saw
"✅ سانس ایجاد شد" either way. Fixed: `publishEvent` now returns
`(invited, groupTitle, posted)` and the confirmation names the target group and
warns when the board did not land.

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
