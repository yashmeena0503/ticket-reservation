A ticket reservation system backend project.

# How to setup dev environment

copy example.env into .env
and alter environment variables to preferences

then run:
```bash
# uses air to automatically restart the app when file changes are detected
docker compose up -d
```

# Migrations
Schema files live in `db/migrations/`,
The files are applied automatically by the `migrate` service in docker on startup.
