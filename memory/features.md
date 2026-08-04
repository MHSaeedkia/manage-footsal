# Features (as actually implemented)

## Private chat

`/start` — registers/updates the user row, then shows an inline main menu.
Any other command replies "invalid command".

Main menu buttons ([internal/bot/bot.go:107](../internal/bot/bot.go#L107)):

| Button | Callback | Who | What it does |
| --- | --- | --- | --- |
| 📝 ثبت نام (register) | `register:<gid>` | not-yet-registered | ask name → ask role → insert `user_groups` |
| ✏️ ویرایش مشخصات (edit) | `edit:<gid>` | registered | same flow, overwrites name + role |
| 💰 صورتحساب (invoice) | `invoice:<gid>` | registered | shows name, role, session count, rate, total debt |
| 💵 تعیین نرخ (set rates) | `set_rates:<gid>` | admin | pick a role → type a number → upsert into `rates` |
| ✅ تسویه حساب کاربر (settle) | `settle:<gid>` | admin | list users → pick one → type session count → subtract |
| 🏟 ایجاد سانس (new event) | `new_event:<gid>` | admin | month → date → capacity, then post board + invite everyone |
| 🔒 بستن سانس (close event) | `close_event:<gid>` | admin | pick an open event → no more answers allowed |
| 📤 ارسال صورتحساب به همه | `bill_all:<gid>` | admin | DM every member their invoice |

Members answer an event from their private chat with `rsvp:present:<eid>` /
`rsvp:absent:<eid>`. See `events-feature.md` for the whole flow and the money rule.

## Group chat

- **Bot added to group** → auto-creates the `groups` row and posts a greeting.
- `/attendance 123456789 987654321` — admin only. Adds **1** session to each user.
  Takes **numeric Bale user IDs** (see `identity-is-numeric-id.md`). Arguments that
  are not valid numbers, IDs with no `users` row, and users who are not members of
  the group are all silently skipped; only a success count is reported.
- `/report` — admin only. Lists every member by their **registered name**
  (`user_groups.name`) with their session balance: `> 0` plain, `< 0` with ❤️,
  `= 0` with ✅.

## Not implemented (despite scaffolding existing)

- **Undo for `/attendance`.** `models.AttendanceRecord` exists with `RevertedAt` /
  `IsReverted` fields, but no migration and no code ever touches it. A mistaken
  `/attendance` still has to be undone by hand with تسویه حساب. (Events *are*
  undoable — a member can flip their own answer until the admin closes it.)
  Its old migration `005_create_attendance_records.sql` was deleted in `95058d9`,
  which burned version 5 forever — see `migration-numbering.md`.
- **Multi-group support in private chat.** `HandleStart` fetches all groups but
  always uses `allGroups[0]` ([internal/handlers/handlers.go:48-52](../internal/handlers/handlers.go#L48-L52)).
  With more than one group it just prints a notice and still uses the first one.
  There is no group picker.
- **Payment / bank integration.** Settlement is a manual number typed by an admin.
