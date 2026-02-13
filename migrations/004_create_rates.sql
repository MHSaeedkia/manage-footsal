-- +goose Up
CREATE TABLE IF NOT EXISTS rates (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    role user_role NOT NULL,
    rate_per_session DECIMAL(10, 2) NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(group_id, role)
);

CREATE INDEX idx_rates_group_id ON rates(group_id);
CREATE INDEX idx_rates_role ON rates(role);

-- +goose Down
DROP TABLE IF EXISTS rates;
