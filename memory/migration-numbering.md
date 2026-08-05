# Goose migration numbers: 005 is burned, never reuse it

Learned the hard way on 2026-08-04.

## The rule

**Before adding a migration, check git history for deleted migration files — not
just the files on disk.**

```bash
git log --all --diff-filter=D --name-only -- migrations/
```

A version number that was ever applied to a live database is permanently taken,
even if the file is gone from the repo.

## What went wrong

`migrations/005_create_attendance_records.sql` existed and was deleted in commit
`95058d9`. The dev database had already run it, so `goose_db_version` holds a row
for **version 5**.

New event migrations were then added as `005_create_events.sql` and
`006_create_event_responses.sql`. On boot goose saw version 5 as already applied,
**silently skipped** the new 005, ran 006, and died:

```
006_create_event_responses.sql: pq: relation "events" does not exist
```

The bot container then crash-looped, because `main.go` calls `zap.L().Fatal` when
migrations fail.

The fix was to renumber to `006_create_events.sql` and
`007_create_event_responses.sql`. Version 5 is left permanently unused; goose does
not mind gaps.

## Why the failure was clean

Goose wraps each SQL migration in a transaction by default, so the failed run
rolled back completely — including the `CREATE TYPE event_response`. Proof: the
container retried many times and failed on the same `CREATE TABLE` every time. If
the type had leaked, the second attempt would have failed earlier with
"type event_response already exists" instead.

## Current state of migration numbers

| Version | File | Note |
| --- | --- | --- |
| 001–004 | users, groups, user_groups, rates | original |
| 005 | *(gone)* | `attendance_records`, deleted in `95058d9`. **Burned.** |
| 006 | `006_create_events.sql` | |
| 007 | `007_create_event_responses.sql` | |
| 008 | `008_create_event_guests.sql` | |

Next new migration starts at **009**.

An old database may still physically have the `attendance_records` table from
version 5. Nothing reads it. Leave it alone unless asked — dropping it needs its
own migration.

Related: [[events-feature]], [[known-issues]]
