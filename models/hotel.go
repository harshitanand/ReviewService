package models

type Hotel struct {
	ID         uint `gorm:"primaryKey"`
	ExternalID int  `gorm:"uniqueIndex"`
	Name       string
}
