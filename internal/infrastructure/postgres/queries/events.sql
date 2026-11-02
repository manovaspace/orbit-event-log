-- name: AppendEvent :exec
INSERT INTO events (stream, event_type, payload, occurred_at)
VALUES ($1, $2, $3, $4);

-- name: AppendEventIdempotent :exec
INSERT INTO events (stream, event_type, payload, occurred_at, idempotency_key)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (stream, idempotency_key) WHERE idempotency_key IS NOT NULL DO NOTHING;

-- name: ReadEvents :many
SELECT id, stream, event_type, payload, occurred_at, COALESCE(idempotency_key, '') AS idempotency_key
FROM events
WHERE stream = $1 AND id > $2
ORDER BY id ASC
LIMIT 1000;
