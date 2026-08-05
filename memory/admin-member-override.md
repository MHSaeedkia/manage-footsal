# Admin sets a member's session status (وضعیت اعضا)

Built on branch `feat-v2`, 2026-08-04. No migration — it reuses `event_responses`.

## What it does

An admin picks a session, picks a member, and sets them حاضر or غایب from private
chat, instead of waiting for that member to tap their own buttons.

Menu: `👥 وضعیت اعضا` → session list → member list → `✅ حاضر` / `❌ غایب`.

Callbacks: `members:<gid>`, `mem_event:<eid>:<gid>`, `mem_pick:<uid>:<eid>:<gid>`,
`mem_set:<present|absent>:<uid>:<eid>:<gid>`.

## Decisions (from the user, 2026-08-04)

- **"Remove" means غایب, not "clear".** There are only the same two states a member
  can set themselves. There is deliberately no "clear the answer" action, so a
  member can never be taken off the board entirely once they are on it.
- **Works on CLOSED sessions too.** Closing stops *members* changing their mind; it
  does not stop an admin fixing a mistake afterwards. This is why the session
  picker here uses `GetEvents` (all sessions, closed marked 🔒) while the close and
  guest pickers use `GetOpenEvents`.
- **The member is told.** They get a private message with
  `ℹ️ ادمین وضعیت شما را برای این سانس ثبت کرد.` on top of their normal session
  bill. Charging someone silently was rejected. The notice is only sent when the
  status actually changed, so re-tapping the same button does not spam them.

## The money rule is shared, on purpose

`handleMemberSetCallback` calls the **same** `sessionDelta` that the member's own
buttons use. An admin override can therefore never charge differently from a
self-service tap. If that rule ever changes, both paths change together — do not
duplicate it.

This also makes the override **idempotent**: setting حاضر on someone who is already
حاضر is a no-op (`delta == 0`). Contrast with `/attendance`, which blindly adds +1
every time it is run.

## Interaction with the other ways to charge a session

There are now three: the member's own buttons, this admin override, and
`/attendance`. The first two share `event_responses` and cannot double-charge each
other. **`/attendance` still knows nothing about events** and will happily add a
second session on top — see the accepted double-charge risk in [[known-issues]].

When in doubt, prefer this override over `/attendance`: it is idempotent, it shows
the current status before you change it, and it updates the group board.

## UI details

- The member list shows every registered member with a mark: `✅` present,
  `❌` absent, `➖` has not answered.
- The session picker is capped at `eventListLimit = 10`, newest first, so it stays
  readable as sessions pile up.
- After a change the admin is returned to the member list, so several people can
  be fixed in a row.

Related: [[events-feature]], [[guests-feature]], [[known-issues]]
