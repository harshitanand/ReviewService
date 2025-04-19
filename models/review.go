package models

import "time"

type Platform struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `gorm:"unique"`
}

type Reviewer struct {
	ID              uint   `gorm:"primaryKey"`
	CountryName     string `gorm:"uniqueIndex:idx_reviewer_identity"`
	ReviewGroupName string `gorm:"uniqueIndex:idx_reviewer_identity"`
	RoomTypeName    string `gorm:"uniqueIndex:idx_reviewer_identity"`
}

type Review struct {
	ID            uint `gorm:"primaryKey"`
	HotelID       uint
	PlatformID    uint
	ReviewerID    uint
	HotelReviewID int64 `gorm:"unique"`
	Rating        float32
	ReviewTitle   string
	ReviewText    string
	ReviewDate    time.Time
	CreatedAt     time.Time
}

type AggregatedHotelReview struct {
	HotelID       uint
	HotelName     string
	AverageRating float32
	ReviewCount   int
}
