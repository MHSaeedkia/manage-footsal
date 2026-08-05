-- +goose Up
CREATE TABLE IF NOT EXISTS event_guests (
    id BIGSERIAL PRIMARY KEY,
    event_id BIGINT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    added_by BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_event_guests_event_id ON event_guests(event_id);

-- +goose Down
DROP TABLE IF EXISTS event_guests;
