# Hotel Booking API

A RESTful hotel room booking API written in Go.

It lets a consumer

- find a hotel by name
- search for rooms available between two dates for a number of guests
- book a room
- look up a booking by its reference.

Built with the Gin framework and PostgreSQL, and it runs with Docker Compose. Double bookings are prevented by the database. Each new booking starts a background goroutine that waits 2 seconds and then logs a message, to simulate sending a confirmation email.

## Use of AI

I used an AI assistant (Claude Code) while building this. It helped with writing and reviewing code, tests, the OpenAPI spec and this README, and with debugging. I reviewed and tested what it produced and made the design decisions myself.

## Quick start

You only need Docker:

```
docker compose up --build
```

Then open http://localhost:9000/docs to try the API in Swagger. Call `POST /v1/admin/seed` first to add test data.

To run the app with Go instead, see [Run locally](#run-locally).

## How a request flows

```
HTTP request -> handler -> service -> store -> Postgres
                   |          |         |
                   |          |         +-- runs the SQL
                   |          +------------ business rules
                   +----------------------- HTTP in, JSON out
```

| Layer | Package | Job |
|---|---|---|
| Handler | `internal/handler` | Reads the request, calls the service, writes the JSON response and picks the HTTP status code |
| Service | `internal/service` | Business rules, such as trimming and validating input and deciding what counts as "not found" |
| Store | `internal/store` | Runs SQL against Postgres using a pgx connection pool |
| Database | `internal/db/migrations` | Schema, applied automatically when the app starts |

Each layer only talks to the one next to it. Errors travel back the same way. For example, the store returns an empty result, the service turns that into "not found", and the handler turns that into a `404`.

Example, `GET /v1/hotels?name=grand`:

1. The handler reads `name` from the URL and calls the service.
2. The service trims it. An empty name stops here with an invalid-input error, which becomes a `400`.
3. The store searches `hotels` for names containing the text, ignoring case.
4. No rows becomes "not found" and a `404`. Otherwise the handler returns `200` with `{"hotels": [...]}`.

## Run locally

You need Go (the version in `go.mod`) and Docker.

1. Start Postgres in a container:

   ```
   docker compose up -d postgres
   ```

2. Run the app:

   ```
   go run ./cmd
   ```

   To change the log level or turn the admin endpoints on or off, set the variables first (see [Settings](#settings)):

   ```
   # Git Bash / macOS / Linux
   LOG_LEVEL=DEBUG ENABLE_ADMIN=true go run ./cmd

   # PowerShell
   $env:LOG_LEVEL = "DEBUG"; $env:ENABLE_ADMIN = "true"; go run ./cmd
   ```

   On startup the app creates the tables itself (it applies the files in `internal/db/migrations`), so there is no separate "create tables" step. The app exits with an error if it can't reach the database.

3. Check that it works:

   ```
   curl -i http://localhost:9000/healthcheck
   ```

   You should get `200` and `{"status":"ok"}`. If Postgres is down, you get `503` and `{"status":"unavailable"}`.

### Try the API in Swagger

Open http://localhost:9000/docs, pick an endpoint, click **Try it out** and then **Execute**. The raw spec is at http://localhost:9000/openapi.yml.

Or use curl, after adding test data (see below):

```
curl "http://localhost:9000/v1/hotels?name=grand"
```

### Settings

The app works without any setup. To change something, set an environment variable:

| Variable | Default | Meaning |
|---|---|---|
| `PORT` | `9000` | Port the API listens on |
| `DATABASE_URL` | `postgres://roomuser:password1@localhost:5433/roomdb?sslmode=disable` | Postgres connection string. The default matches `docker-compose.yml` |
| `LOG_LEVEL` | `INFO` | `DEBUG`, `INFO`, `WARN` or `ERROR` |
| `ENABLE_ADMIN` | `true` | `true` or `false`. Turns on `POST /v1/admin/seed` (add test data) and `POST /v1/admin/reset` (delete all data). Set it to `false` anywhere real data lives. The app won't start if the value isn't a valid true/false |

### Add test data

Start the app first, so the tables exist. Then:

```
curl -X POST http://localhost:9000/v1/admin/seed
```

This adds 3 hotels with 6 rooms each, and a few bookings. It is safe to call more than once. You can also do it from Swagger.

### Remove test data

Removes all data and restarts ids, ready for seeding. The tables are kept.

```
curl -X POST http://localhost:9000/v1/admin/reset
```

Both endpoints can be turned off with `ENABLE_ADMIN=false`, which is what you would do anywhere real data lives. The SQL they run is in `internal/db/seeds/`.

### Start again from an empty database

The simplest way is to delete the database volume. This removes all data. The next `go run ./cmd` creates the tables again.

```
docker compose down -v
docker compose up -d postgres
```

Please don't run `internal/db/migrations/000001_schema.down.sql` by hand to do this. It drops the tables but leaves the app's `schema_migrations` record saying they exist, so the app would skip creating them again.

### Database admin UI (optional)

`docker compose up -d` also starts the app and pgAdmin. pgAdmin is at http://localhost:5050 (login `admin@example.com` / `admin`, local use only). Postgres is on port `5433` from your machine, and `5432` from inside Docker.

## Before you push

Run these from the project root, in this order. They are the same checks CI runs, so if they pass here they should pass there.

```
go mod tidy
go mod vendor
go vet ./...
golangci-lint run ./...
go test ./...
```

What each one is for:

| Command | Why |
|---|---|
| `go mod tidy` | Removes unused dependencies and adds missing ones. CI fails if this changes `go.mod` or `go.sum`, so commit those files if they change. |
| `go mod vendor` | Refreshes the `vendor/` folder to match `go.mod`. Run it whenever you add or remove a dependency, and commit the result. |
| `go vet ./...` | Catches suspicious code that compiles but is probably wrong. |
| `golangci-lint run ./...` | Runs the linters. It only reports problems and doesn't change your files. On Windows the command is `golangci-lint.exe`, though plain `golangci-lint` usually works too. |
| `go test ./...` | Runs the unit tests. The store tests are skipped unless a database is set up, see [Tests](#tests). |

Install the linter with the same version CI uses (`v2.14.0` in `ci.yml`):

```
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0
```

## Tests

The tests are kept small on purpose. Right now they cover booking a room (`POST /v1/bookings`) in each layer:

| Layer | File | What it checks |
|---|---|---|
| Handler (unit) | `internal/handler/handler_test.go` | A new booking gives `201` with a `Location` header. A missing `Idempotency-Key` gives `400`. A room that is already booked gives `409`. Uses a fake service. |
| Service (unit) | `internal/service/service_test.go` | A booking is created and saved. Too many guests, a check-in in the past, and a room that is already booked are rejected. Uses a fake store. |
| Store (integration) | `internal/store/postgres_test.go` | Against a real Postgres: a booking is saved, an overlapping stay is refused, and a repeated reference is refused. |

Unit tests need nothing else:

```
go test ./...
```

The store test needs Postgres. It builds the real schema in a temporary schema and drops it afterwards, so your dev data is not touched:

```
docker compose up -d postgres
TEST_DATABASE_URL="postgres://roomuser:password1@localhost:5433/roomdb?sslmode=disable" go test ./internal/store/
```

If `TEST_DATABASE_URL` is not set, the store test is skipped, so `go test ./...` still passes on a machine without a database.

## CI pipeline

Every push to `master` and every pull request runs the GitHub Actions workflow in `.github/workflows/ci.yml`.

```
lint ─┐
build ┼─> test (integration)
test ─┘
```

`test` here is the unit test job. The three jobs on the left are quick and don't depend on each other, so they start at the same time.

### How it works

- **Cheap checks first.** Lint, build and unit tests are fast. They run in parallel and fail fast, so if you break something you see all the problems in one go, not one at a time.
- **Integration waits for them.** The integration job starts a real Postgres container, which is the slowest and most expensive part. It only runs once lint, build and the unit tests have passed. There is no point paying for a database on a commit that doesn't even compile.

### What each job does

| Job | What it does |
|---|---|
| `lint` | Runs `golangci-lint` |
| `build` | Checks `go mod tidy` leaves no changes, verifies modules, and builds everything |
| `test` | Runs `go vet` and the unit tests with `-race` |
| `test (integration)` | Starts Postgres 17 and runs the store tests against it |

### Why not one long chain?

We could run `build -> lint -> test -> integration` one after another, but it would be slower and wouldn't catch anything extra. These jobs don't use each other's output, so waiting on each other only adds time. Also, if lint failed early, you wouldn't hear about a failing test until the next push.

A strict chain makes sense when a job really needs what the previous one made, for example a deploy job that uses a Docker image built earlier. We don't have that yet.

### A trap we guard against

The store tests skip themselves when `TEST_DATABASE_URL` is not set, which is handy on a laptop with no database. In CI that would be dangerous: if the variable got lost, the job would go green without testing anything. So the integration job fails if any test was skipped.
