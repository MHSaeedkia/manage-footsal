-- +goose Up
-- Guests added before this column existed were all charged, so FALSE is the
-- correct value for every existing row.
ALTER TABLE event_guests ADD COLUMN IF NOT EXISTS is_free BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE event_guests DROP COLUMN IF EXISTS is_free;
