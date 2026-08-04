-- +goose Up
CREATE TYPE event_response AS ENUM ('present', 'absent');

CREATE TABLE IF NOT EXISTS event_responses (
    id BIGSERIAL PRIMARY KEY,
    event_id BIGINT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    response event_response NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(event_id, user_id)
);

CREATE INDEX idx_event_responses_event_id ON event_responses(event_id);
CREATE INDEX idx_event_responses_user_id ON event_responses(user_id);

-- +goose Down
DROP TABLE IF EXISTS event_responses;
DROP TYPE IF EXISTS event_response;
