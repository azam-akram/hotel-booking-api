# Hotel Booking API

A RESTful hotel room booking API written in Go. 

It lets 

- a consumer find a hotel by name
- search for rooms available between two dates for a number of guests
- book a room
- look up a booking by its reference.

Built with Gin framework and PostgreSQL, and it runs locally with Docker Compose. Double bookings are prevented by the database, and each booking triggers a simulated confirmation email (e.g., logging a message
to the console after a 2-second delay using a goroutine) in a background goroutine.

## run locally

run postgresql db in container
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