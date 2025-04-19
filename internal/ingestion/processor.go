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

func safeWrapJLLine(raw map[string]interface{}, db *gorm.DB) error {
	defer func() {
		if r := recover(); r != nil {
			panic(r)
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

	// ✅ Ensure hotel creation is concurrency-safe
	var hotel models.Hotel
	if err := db.Where("external_id = ?", hotelID).First(&hotel).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			newHotel := models.Hotel{ExternalID: hotelID, Name: hotelName}
			if err := db.Create(&newHotel).Error; err != nil {
				// Retry after create error (duplicate from another thread)
				if dbErr := db.Where("external_id = ?", hotelID).First(&hotel).Error; dbErr != nil {
					log.Printf("❌ Hotel fetch failed after duplicate insert (hotelId=%d): %v", hotelID, dbErr)
					return
				}
			} else {
				hotel = newHotel
			}
		} else {
			log.Printf("❌ DB error querying hotel (hotelId=%d): %v", hotelID, err)
			return
		}
	}

	// ✅ Platform creation (also concurrency-safe)
	var platform models.Platform
	if err := db.Where("name = ?", platformName).First(&platform).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			platform = models.Platform{Name: platformName}
			if err := db.Create(&platform).Error; err != nil {
				if err := db.Where("name = ?", platformName).First(&platform).Error; err != nil {
					log.Printf("❌ Platform recovery failed: %v", err)
					return
				}
			}
		} else {
			log.Printf("❌ DB error querying platform: %v", err)
			return
		}
	}

	// ✅ Reviewer creation (based on composite key)
	reviewer := models.Reviewer{
		CountryName:     getStr(reviewerInfo["countryName"]),
		ReviewGroupName: getStr(reviewerInfo["reviewGroupName"]),
		RoomTypeName:    getStr(reviewerInfo["roomTypeName"]),
	}
	if err := db.Where(&reviewer).FirstOrCreate(&reviewer).Error; err != nil {
		log.Printf("❌ Error creating reviewer: %v", err)
		return
	}

	// ✅ Review creation (skip if duplicate)
	hotelReviewID := parseInt64(comment["hotelReviewId"])
	var existing models.Review
	if err := db.Where("hotel_review_id = ?", hotelReviewID).First(&existing).Error; err == nil {
		return // duplicate
	} else if err != gorm.ErrRecordNotFound {
		log.Printf("❌ Error checking for review (id=%d): %v", hotelReviewID, err)
		return
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

	// ✅ Rating summary update
	var summary models.HotelRatingsSummary
	if err := db.Where("hotel_id = ?", hotel.ID).First(&summary).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			summary = models.HotelRatingsSummary{
				HotelID:       hotel.ID,
				TotalReviews:  1,
				TotalRating:   float64(review.Rating),
				AverageRating: float64(review.Rating),
				LastUpdated:   time.Now(),
			}
			db.Create(&summary)
		} else {
			log.Printf("❌ Error loading hotel summary: %v", err)
		}
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
