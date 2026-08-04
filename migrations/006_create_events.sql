-- +goose Up
CREATE TABLE IF NOT EXISTS events (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    created_by BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    month VARCHAR(255) NOT NULL,
    session_date VARCHAR(255) NOT NULL,
    capacity INTEGER NOT NULL,
    group_message_id INTEGER,
    is_closed BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_events_group_id ON events(group_id);
CREATE INDEX idx_events_is_closed ON events(is_closed);

-- +goose Down
DROP TABLE IF EXISTS events;
