CREATE TABLE events (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organizer_id UUID NOT NULL REFERENCES users(id),
    name         TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    venue        TEXT NOT NULL,
    starts_at    TIMESTAMPTZ NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE sections (
    id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    name     TEXT NOT NULL,
    price    INTEGER NOT NULL CHECK (price >= 0),
    UNIQUE   (event_id, name)
);

CREATE TABLE seats (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    section_id  UUID NOT NULL REFERENCES sections(id) ON DELETE CASCADE,
    row_label   TEXT NOT NULL,
    seat_number INTEGER NOT NULL CHECK (seat_number > 0),
    status      TEXT NOT NULL DEFAULT 'available'
    CHECK       (status IN ('available', 'held', 'booked')),
    UNIQUE      (section_id, row_label, seat_number)
);
