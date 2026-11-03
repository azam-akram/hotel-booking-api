package model

type RoomType string

const (
	RoomTypeSingle RoomType = "single"
	RoomTypeDouble RoomType = "double"
	RoomTypeDeluxe RoomType = "deluxe"
)

type Hotel struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type HotelListResponse struct {
	Hotels []Hotel `json:"hotels"`
}

type Room struct {
	ID       int64    `json:"id"`
	HotelID  int64    `json:"-"`
	Number   string   `json:"number"`
	Type     RoomType `json:"type"`
	Capacity int      `json:"capacity"`
}
