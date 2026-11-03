package model

type Hotel struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type HotelListResponse struct {
	Hotels []Hotel `json:"hotels"`
}
