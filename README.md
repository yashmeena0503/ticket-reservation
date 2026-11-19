# Ticket Reservation API

REST API in Go for event ticketing. Organizers create events and seating sections, and seats are generated for them automatically. Built with Go's standard library, PostgreSQL and Docker Compose, with an end-to-end test suite.

## Features

- **Auth:** Argon2id password hashing and short-lived login tokens.
- **Access control:** only an event's organizer can add sections to it.
- **Data integrity:** the database enforces its own rules, and a section is never saved without its seats.
- **Pagination:** event listing stays stable and fast as the number of events grows.
- **Operations:** structured logs, request tracing, panic recovery, health and readiness checks, graceful shutdown.
- **Dev experience:** hot reload with [air](https://github.com/air-verse/air) inside Docker, migrations applied automatically on startup.

## Quick start

```bash
cp example.env .env             # edit with preferences
docker compose up -d api-dev    # starts db, runs migrations, then the API with hot reload
curl localhost:3000/ready
```

## API

| Method | Path                  | Auth      | Description                                  |
| ------ | --------------------- | --------- | -------------------------------------------- |
| GET    | `/health`             |           | Liveness                                     |
| GET    | `/ready`              |           | Readiness (pings the database)               |
| POST   | `/auth/register`      |           | Create a user                                |
| POST   | `/auth/login`         |           | Returns `access_token`                       |
| GET    | `/auth/me`            | Bearer    | Current user id                              |
| GET    | `/event`              |           | List events, keyset-paginated                |
| GET    | `/event/{id}`         |           | Event with sections and seat availability    |
| POST   | `/event`              | Bearer    | Create an event                              |
| POST   | `/event/{id}/section` | Organizer | Create a section and generate its seats      |

## Testing

End-to-end tests are [k6](https://k6.io) scripts in `tests/`, run against the live `api-dev` container through Compose.

```bash
make test          # lists available tests
make test static   # runs tests/static.js; fails if any check fails
```

`static.js` covers auth (register, duplicate, wrong password, token checks), validation, organizer-only authorization, pagination, and seat counts.

## Project layout

```
cmd/api/                 entrypoint, wiring, graceful shutdown
internal/auth/           register, login, JWT issue/verify
internal/events/         events and sections handlers
internal/db/             sqlc-generated code (do not edit)
internal/platform/       config, pgx pool + tx helper, HTTP middleware and JSON helpers
db/migrations/           SQL schema, applied by the `migrate` service
db/queries/              sqlc query definitions
tests/                   k6 scripts
```

Regenerate the query code after editing `db/queries/` or `db/migrations/`:

```bash
sqlc generate
```
