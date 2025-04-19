package ingestion

import (
	"fmt"
	"log"
	"review-system/models"
	"strings"
	"time"

	"gorm.io/gorm"
)

func ProcessJLLineWithSuppressedErrors(raw map[string]interface{}, db *gorm.DB) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("⚠️  Recovered from panic while processing: %v", r)
		}
	}()

	err := safeWrapJLLine(raw, db)
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Printf("❌ Error processing review: %v", err)
	}
}

// safeWrapJLLine wraps the original ProcessJLLine to return errors
func safeWrapJLLine(raw map[string]interface{}, db *gorm.DB) error {
	defer func() {
		if r := recover(); r != nil {
			panic(r) // Let the outer recover handle it
		}
	}()

	ProcessJLLine(raw, db)
	return nil
}

func ProcessJLLine(raw map[string]interface{}, db *gorm.DB) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in ingestion: %v", r)
		}
	}()

	hotelID := int(raw["hotelId"].(float64))
	platformName := raw["platform"].(string)
	hotelName := raw["hotelName"].(string)

	comment := raw["comment"].(map[string]interface{})
	reviewerInfo := comment["reviewerInfo"].(map[string]interface{})

	hotel := models.Hotel{ExternalID: hotelID, Name: hotelName}
	db.FirstOrCreate(&hotel, models.Hotel{ExternalID: hotelID})

	platform := models.Platform{Name: platformName}
	db.FirstOrCreate(&platform, platform)

	reviewer := models.Reviewer{
		CountryName:     getStr(reviewerInfo["countryName"]),
		ReviewGroupName: getStr(reviewerInfo["reviewGroupName"]),
		RoomTypeName:    getStr(reviewerInfo["roomTypeName"]),
	}
	db.FirstOrCreate(&reviewer, reviewer)

	hotelReviewID := parseInt64(comment["hotelReviewId"])

	var existing models.Review
	err := db.Where("hotel_review_id = ?", hotelReviewID).First(&existing).Error
	if err == nil {
		// Found a duplicate, skip insertion
		return
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		// Log real errors only
		log.Printf("❌ DB error while checking for existing review (hotelReviewID=%d): %v", hotelReviewID, err)
	}

	review := models.Review{
		HotelID:       hotel.ID,
		PlatformID:    platform.ID,
		ReviewerID:    reviewer.ID,
		HotelReviewID: hotelReviewID,
		Rating:        float32(comment["rating"].(float64)),
		ReviewTitle:   getStr(comment["reviewTitle"]),
		ReviewText:    getStr(comment["reviewComments"]),
		ReviewDate:    parseTime(getStr(comment["reviewDate"])),
	}

	if err := db.Create(&review).Error; err != nil {
		log.Printf("❌ Failed to insert review (hotelReviewId=%d): %v", hotelReviewID, err)
		return
	}

	var summary models.HotelRatingsSummary
	if err := db.First(&summary, "hotel_id = ?", hotel.ID).Error; err != nil {
		summary = models.HotelRatingsSummary{
			HotelID:       hotel.ID,
			TotalReviews:  1,
			TotalRating:   float64(review.Rating),
			AverageRating: float64(review.Rating),
			LastUpdated:   time.Now(),
		}
		db.Create(&summary)
	} else {
		summary.TotalReviews++
		summary.TotalRating += float64(review.Rating)
		summary.AverageRating = summary.TotalRating / float64(summary.TotalReviews)
		summary.LastUpdated = time.Now()
		db.Save(&summary)
	}
}

func getStr(val interface{}) string {
	if val == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", val))
}

func parseInt64(val interface{}) int64 {
	switch v := val.(type) {
	case float64:
		return int64(v)
	case string:
		var parsed int64
		fmt.Sscanf(v, "%d", &parsed)
		return parsed
	default:
		return 0
	}
}

func parseTime(val string) time.Time {
	t, _ := time.Parse(time.RFC3339, val)
	return t
}
