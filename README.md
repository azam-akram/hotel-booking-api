# Hotel Booking API

A RESTful hotel room booking API written in Go. 

It lets 

- a consumer find a hotel by name
- search for rooms available between two dates for a number of guests
- book a room
- look up a booking by its reference.

Built with Gin framework and PostgreSQL, and it runs locally with Docker Compose. Double bookings are prevented by the database, and each booking triggers a simulated confirmation email (e.g., logging a message
to the console after a 2-second delay using a goroutine) in a background goroutine.

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

## run locally

Run postgresql db in docker container
```
docker compose up -d postgres
```

and then run app

```
go run ./cmd
```

check health
```
curl -i http://localhost:9000/healthcheck
```

## Create the tables
```
docker compose exec -T postgres psql -U roomuser -d roomdb -v ON_ERROR_STOP=1 < internal/db/migrations/000001_schema.up.sql
```

##  Add test data

```
docker compose exec -T postgres psql -U roomuser -d roomdb -v ON_ERROR_STOP=1 < internal/db/seeds/seed.sql
```

##  Remove test data

Removes all data and restarts ids, ready for seeding. The schema is kept.

```
docker compose exec -T postgres psql -U roomuser -d roomdb -v ON_ERROR_STOP=1 < internal/db/seeds/reset.sql
```

## Drop database schema
```
$ docker compose exec -T postgres psql -U roomuser -d roomdb -v ON_ERROR_STOP=1 < internal/db/migrations/000001_schema.down.sql
```

## Tests

Right now the tests cover the "find a hotel by name" feature (`GET /v1/hotels?name=`). The feature has three layers, and each layer has its own test. Every test checks one thing the other layers can't.

```
HTTP request -> handler -> service -> store -> Postgres
```

### Run them

Unit tests (no database needed):

```
go test ./...
```

Integration test for the store (needs Postgres):

```
docker compose up -d postgres
TEST_DATABASE_URL="postgres://roomuser:password1@localhost:5433/roomdb?sslmode=disable" go test ./internal/store/
```

If `TEST_DATABASE_URL` is not set, the store tests are skipped, so `go test ./...` still passes on a machine without a database.

### Handler tests (unit)

File: `internal/handler/handler_test.go`

The handler is the thin layer between HTTP and our code. It reads the query, calls the service, and turns the result into JSON or an error response. The test uses a fake service, so no real logic or database is involved.

What it checks:

- a good result gives `200` and `{"hotels": [...]}`
- the `name` from the URL reaches the service untouched
- an "invalid input" error gives `400`, "not found" gives `404`, and anything unexpected gives `500`
- a `500` shows a generic message, so internal details like a database address never leak to the caller

Why: if someone changes how errors map to status codes, we want to know straight away. These tests are fast and never flaky.

### Service tests (unit)

File: `internal/service/service_test.go`

The service holds the business rules. The test uses a fake store, so again no database.

What it checks:

- spaces around the name are trimmed (`"  Grand Palace "` becomes `"Grand Palace"`)
- an empty or blank name is rejected as invalid, and the store is never called
- an empty or `nil` result from the store becomes a "not found" error
- a store error is passed back as it is

Why: these are the rules of the feature. The "store is never called" checks matter too. A bad request shouldn't cost a database trip.

### Store test (integration)

File: `internal/store/postgres_test.go`

The store runs the SQL. A fake database here would only prove that our fake works, so this test talks to a real Postgres.

It does not touch your dev data. Each run creates a temporary schema with a small `hotels` table, runs the test inside it, and drops it afterwards.

What it checks:

- exact match, and matching that ignores upper and lower case
- partial match (`grand` finds `Grand Palace Hotel`), sorted by name
- an exact match comes before longer names that only contain the text
- `%` and `_` in the search text are plain characters, not wildcards (that is why the query uses `strpos` and not `LIKE`)
- no match returns an empty list, so the JSON is `[]` and not `null`
- a closed connection returns an error

Why: the SQL is the part most likely to be wrong, and only a real database can tell us. Keeping it separate from the unit tests means the fast tests stay fast.
