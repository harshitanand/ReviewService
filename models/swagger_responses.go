package models

type ErrorResponse struct {
	Error string `json:"error"`
}

type ReviewDetail struct {
	Rating          float32 `json:"rating"`
	ReviewTitle     string  `json:"review_title"`
	ReviewText      string  `json:"review_text"`
	ReviewDate      string  `json:"review_date"`
	CountryName     string  `json:"country_name"`
	ReviewGroupName string  `json:"review_group_name"`
	RoomTypeName    string  `json:"room_type_name"`
}

type ReviewResponse struct {
	Hotel   AggregatedHotelReview `json:"hotel"`
	Reviews []ReviewDetail        `json:"reviews"`
}
