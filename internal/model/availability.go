package model

type AvailabilityQuery struct {
	CheckIn  Date `form:"check_in" binding:"required"`
	CheckOut Date `form:"check_out" binding:"required"`
	Guests   int  `form:"guests" binding:"min=1,max=10"`
}

type AvailableRoomsResponse struct {
	HotelID  int64  `json:"hotel_id"`
	CheckIn  Date   `json:"check_in"`
	CheckOut Date   `json:"check_out"`
	Nights   int    `json:"nights"`
	Guests   int    `json:"guests"`
	Rooms    []Room `json:"rooms"`
}
