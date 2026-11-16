-- name: NewEvent :one
INSERT INTO events (organizer_id, name, description, venue, starts_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetEvent :one
SELECT * FROM events WHERE id = $1;

-- name: ListEventsAfter :many
SELECT * FROM events
WHERE (starts_at, id) > (@after_starts_at::timestamptz, @after_id::uuid)
ORDER BY starts_at, id
LIMIT @page_size;

-- name: NewSection :one
INSERT INTO sections (event_id, name, price)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListSectionsWithAvailability :many
SELECT s.id, s.event_id, s.name, s.price,
count(st.id)::int AS total_seats,
count(st.id) FILTER (WHERE st.status = 'available')::int AS available_seats
FROM sections s
LEFT JOIN seats st ON st.section_id = s.id
WHERE s.event_id = $1
GROUP BY s.id
ORDER BY s.name;

-- name: NewSeat :copyfrom
INSERT INTO seats (section_id, row_label, seat_number)
VALUES ($1, $2, $3);
