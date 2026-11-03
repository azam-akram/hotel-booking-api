CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE TYPE room_type AS ENUM ('single', 'double', 'deluxe');

CREATE TABLE hotels (
    id   BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name TEXT NOT NULL CHECK (length(name) BETWEEN 1 AND 100)
);

CREATE UNIQUE INDEX hotels_name_lower_key ON hotels (lower(name));

CREATE TABLE rooms (
    id       BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    hotel_id BIGINT    NOT NULL REFERENCES hotels (id) ON DELETE CASCADE,
    number   TEXT      NOT NULL,
    type     room_type NOT NULL,
    capacity INT       NOT NULL CHECK (capacity > 0),
    CONSTRAINT rooms_hotel_number_key UNIQUE (hotel_id, number)
);

CREATE TABLE bookings (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    reference       TEXT        NOT NULL,
    room_id         BIGINT      NOT NULL REFERENCES rooms (id) ON DELETE CASCADE,
    guest_name      TEXT        NOT NULL,
    guest_email     TEXT        NOT NULL,
    check_in        DATE        NOT NULL,
    check_out       DATE        NOT NULL,
    guests          INT         NOT NULL CHECK (guests > 0),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    idempotency_key TEXT,
    request_hash    TEXT,

    CONSTRAINT bookings_reference_key UNIQUE (reference),
    CONSTRAINT bookings_idempotency_key_key UNIQUE (idempotency_key),
    CONSTRAINT bookings_stay_check CHECK (check_out > check_in),

    CONSTRAINT bookings_no_overlap
        EXCLUDE USING gist (room_id WITH =, daterange(check_in, check_out, '[)') WITH &&)
);
