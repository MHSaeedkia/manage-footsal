# Futsal events (sessions with RSVP)

Built on branch `feat-v2`, 2026-08-04. This is the main way sessions get charged now.

## Flow

1. Admin taps **🏟 ایجاد سانس** in the private menu.
2. Bot asks three things in order, one message each:
   `month` (var1) → `session_date` (var2) → `capacity` (var3, must be a positive int).
3. Bot posts the board template in the group and **saves that message's ID** in
   `events.group_message_id`.
4. Bot sends every registered member of the group a private invite with two
   buttons: **✅ میام** / **❌ نمیام**.
5. Each answer re-renders and edits the group board in place.
6. After each answer the member gets a private bill for that session.

## The three variables (confirmed by the user)

| Var | Column | Type | Meaning |
| --- | --- | --- | --- |
| var1 | `month` | string | month name, e.g. `مرداد` — rendered as `*مرداد ماه*` |
| var2 | `session_date` | string | date, e.g. `جمعه ۱۵` |
| var3 | `capacity` | int | how many people fit |

**Do not confuse var1 and var2.** var1 is the month, var2 is the date. The template
line `حاضرین و غایبین سالن *var2* : فوتسال` looks scrambled in an editor because it
is RTL text with an LTR placeholder — that is a display artifact, the stored order
is correct.

## Capacity is display only

The user was asked what should happen when the present list passes `capacity` and
answered **"add them"**. So `capacity` is printed in the board and nothing else —
there is no limit, no rejection, no waiting list. Do not add enforcement unless
asked.

## The money rule

Saying **present is exactly one `/attendance`**: `sessions_owed += 1`. Changing the
answer moves it back. This lives in `sessionDelta` in
`internal/handlers/handlers_event.go` and is covered by tests:

| previous | new answer | delta |
| --- | --- | --- |
| (none) | present | +1 |
| (none) | absent | 0 |
| present | present | 0 |
| absent | absent | 0 |
| present | absent | -1 |
| absent | present | +1 |

The "same answer → 0" rows are what stop double-charging when someone taps the same
button twice. `TestSessionDeltaDoesNotDrift` pins this.

## Closing

Members can change their answer **until an admin closes the event** (this was the
user's choice). Closing is **🔒 بستن سانس** → pick from the open list. After that
`rsvp` callbacks are refused and the board gets a `🔒 اعلام حضور بسته شد.` line.

## Markdown escaping matters here

The board is sent with `ParseMode: Markdown` and contains user-typed names. An
unescaped `*` or `_` in a name makes the **whole message fail to parse**, which
means the board silently stops updating for everyone. `escapeMarkdown` in
`handlers_event.go` handles `_ * ` + backtick + `[`. Any new user text put into a
Markdown message must go through it.

(This is the legacy-Markdown escaper. The old unused `escapeMarkdownV2` in
`handlers_admin.go` is for Markdown**V2** and escapes far more characters — using it
here would print literal backslashes. They are not interchangeable.)

## Known gap the user accepted

`/attendance` was **kept** alongside events (the user chose this). Nothing links the
two, so a person can be charged twice for one session: once by tapping ✅ میام and
once by being named in `/attendance`. There is no detection for it. The admin has
to fix it manually with تسویه حساب.

Related: [[identity-is-numeric-id]], [[known-issues]], [[features]]
