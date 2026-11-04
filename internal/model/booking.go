package model

import "time"

type Booking struct {
	Reference  string    `json:"reference"`
	HotelID    int64     `json:"hotel_id"`
	Room       Room      `json:"room"`
	GuestName  string    `json:"guest_name"`
	GuestEmail string    `json:"guest_email"`
	CheckIn    Date      `json:"check_in"`
	CheckOut   Date      `json:"check_out"`
	Nights     int       `json:"nights"`
	Guests     int       `json:"guests"`
	CreatedAt  time.Time `json:"created_at"`
}

type BookingRequest struct {
	HotelID    int64  `json:"hotel_id" binding:"gt=0"`
	RoomID     int64  `json:"room_id" binding:"gt=0"`
	GuestName  string `json:"guest_name" binding:"required,min=1,max=100"`
	GuestEmail string `json:"guest_email" binding:"required,email,max=254"`
	CheckIn    Date   `json:"check_in" binding:"required"`
	CheckOut   Date   `json:"check_out" binding:"required"`
	Guests     int    `json:"guests" binding:"min=1,max=10"`
}

type BookingResponse struct {
	Booking  Booking
	Replayed bool
}

// Idempotency stores the key used to identify a booking request.
// RequestHash checks whether a request with the same key has the same data.
// If the key is empty, idempotency is not used.
type Idempotency struct {
	Key         string
	RequestHash string
}
