FROM golang:1.27-alpine AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -o /hotel-booking-api ./cmd

FROM alpine:3.21
COPY --from=build /hotel-booking-api /hotel-booking-api
EXPOSE 9000
ENTRYPOINT ["/hotel-booking-api"]
