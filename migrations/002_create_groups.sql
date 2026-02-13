-- +goose Up
CREATE TABLE IF NOT EXISTS groups (
    id BIGSERIAL PRIMARY KEY,
    telegram_chat_id BIGINT NOT NULL UNIQUE,
    title VARCHAR(255),
    type VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_groups_telegram_chat_id ON groups(telegram_chat_id);

-- +goose Down
DROP TABLE IF EXISTS groups;
