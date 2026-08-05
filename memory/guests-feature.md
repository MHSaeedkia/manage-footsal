# Guests (people with no account)

Built on branch `feat-v2`, 2026-08-04. Migration **008** (`event_guests`).

## What a guest is

Someone who plays but is not a user of the bot. A **registered member** types their
name in private chat and it appears in the group board's present list marked
`(مهمان)`.

- **One session only.** `event_guests` rows point at one `event_id`. Next session
  starts empty; the same person must be added again.
- **Whoever adds the guest pays for them.** Adding a guest is `sessions_owed += 1`
  on the adder's `user_groups` row. Removing gives it back (`-1`).
- **Charged at the adder's own role rate.** There is deliberately **no guest role
  and no guest price** — see the decision below.
- **Removable**, by the person who added them, or by any admin.

## Decision: no separate guest role (2026-08-04)

The user first asked for a `guest` role in the `user_role` enum with its own price.
That was raised as a problem: `sessions_owed` is a single integer and money is
always `sessions_owed × rate(role)`, so a second price needs a second counter,
which then needs settlement and `/report` to handle two kinds of session.

The user's answer was **"let it go, i do not need new user, each user add a guest
should pay by its price role"** — so:

- no new enum value, **no migration to `user_role`**
- no `guest_sessions_owed` column
- `/report`, تسویه حساب and the invoice are all **unchanged** — a guest is simply
  one more ordinary session on the adder's account

If guest pricing ever comes back, it needs the second counter *and* a settlement
design. Do not add a `guest` enum value on its own; it would silently bill at the
wrong rate.

## Who can do what

| Action | Who |
| --- | --- |
| Add a guest | any member **registered in that group** |
| Remove own guest | the member who added it |
| Remove anyone's guest | admins (refund still goes to the original adder) |

Membership is enforced by `guestMember` in `handlers_guest.go`. This matters: the
charge is an `UPDATE user_groups ... WHERE user_id = ...`, so a caller with no
`user_groups` row would match **zero rows** and get their guest for free. A default
admin who never registered in the group therefore cannot manage guests — that is
intentional, not a bug.

## Money safety

`AddEventGuest` and `DeleteEventGuest` each run the guest row and the money change
in **one transaction**, so the board can never disagree with the ledger. This is
the only place in the repo using an explicit transaction — the rest of the
repository does single statements.

`DeleteEventGuest` uses `DELETE ... RETURNING added_by`. That is what makes a
double tap safe: the second call deletes no row, returns `sql.ErrNoRows`, and no
second refund happens. It also finds the correct person to refund even when an
admin removes someone else's guest.

## Admin flow / callbacks

`🧑‍🤝‍🧑 مهمان سانس` (now in the **member** part of the main menu, not the admin part)
→ pick an open session → `➕ افزودن مهمان` / `❌ حذف <name>` / `🔙 بازگشت`.

Callbacks: `guests:<gid>`, `guest_menu:<eid>:<gid>`, `add_guest:<eid>:<gid>`,
`del_guest:<guest_id>:<eid>:<gid>`. State: `awaiting_guest_name`.

## Board rendering

Guests are appended **after** the members inside حاضرین, numbering running across
both. They never appear in غایبین.

```
🔹 حاضرین :
1- علی
2- سارا
3- رضا (مهمان)
🔻 غایبین:
1- حسین
```

`buildEventBoard(event, answers, guests)` is pure and tested, including
guests-only sessions and Markdown escaping of guest names.

## Gotchas

- Guests attach only to **open** sessions (`GetOpenEvents` feeds the picker).
- Duplicate guest names in one session are allowed; removal is by row id.
- Guests do not count against `capacity` — nothing does, it is display-only.
- Because money is always `sessions × current rate`, changing a member's role or
  rate later also re-prices the guests they already added. That is pre-existing
  behaviour of the whole ledger, not specific to guests.
- **The guest money path has no automated test** — it is SQL in a transaction and
  would need a live database.

Related: [[events-feature]], [[migration-numbering]]
