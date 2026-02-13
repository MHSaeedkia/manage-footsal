-- +goose Up
CREATE TABLE IF NOT EXISTS attendance_records (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    admin_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_ids BIGINT[] NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    reverted_at TIMESTAMP WITH TIME ZONE,
    is_reverted BOOLEAN DEFAULT FALSE
);

CREATE INDEX idx_attendance_records_group_id ON attendance_records(group_id);
CREATE INDEX idx_attendance_records_created_at ON attendance_records(created_at);
CREATE INDEX idx_attendance_records_is_reverted ON attendance_records(is_reverted);

-- +goose Down
DROP TABLE IF EXISTS attendance_records;
