-- name: NewEvent :one
INSERT INTO events (organizer_id, name, description, venue, starts_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListEventsAfter :many
SELECT * FROM events
WHERE (starts_at, id) > (@after_starts_at::timestamptz, @after_id::uuid)
ORDER BY starts_at, id
LIMIT @page_size;
