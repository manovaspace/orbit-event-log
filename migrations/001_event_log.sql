-- +goose Up
CREATE TABLE IF NOT EXISTS events (
    id BIGSERIAL PRIMARY KEY,
    stream TEXT NOT NULL,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    idempotency_key TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_events_stream_idempotency
    ON events (stream, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_events_stream_id ON events (stream, id);

-- +goose Down
DROP TABLE IF EXISTS events;
