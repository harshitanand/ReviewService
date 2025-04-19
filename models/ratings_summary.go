package models

import "time"

type HotelRatingsSummary struct {
	HotelID       uint    `gorm:"primaryKey"`
	TotalReviews  int     `gorm:"default:0"`
	TotalRating   float64 `gorm:"default:0"`
	AverageRating float64
	LastUpdated   time.Time
}
