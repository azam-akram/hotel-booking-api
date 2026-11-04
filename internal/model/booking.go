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
