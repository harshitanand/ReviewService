package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"review-system/models"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

func IngestJLFileAsync(filename string) {
	go func() {
		if err := ingestJLWorkerPool(filename); err != nil {
			log.Printf("❌ Ingestion failed: %v", err)
		} else {
			log.Println("✅ Background ingestion completed")
		}
	}()
}

func ingestJLWorkerPool(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	const workerCount = 8
	db := models.GetDB()

	lines := make(chan map[string]interface{}, 100)
	var wg sync.WaitGroup

	// Spawn workers
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for raw := range lines {
				processJLLine(raw, db)
			}
		}()
	}

	// Feed lines to workers
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var raw map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &raw); err == nil {
			lines <- raw
		}
	}
	close(lines)
	wg.Wait()

	return nil
}

func processJLLine(raw map[string]interface{}, db *gorm.DB) {
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

	// Prevent duplicates
	var existing models.Review
	if err := db.Where("hotel_review_id = ?", hotelReviewID).First(&existing).Error; err == nil {
		// log.Printf("⚠️  Duplicate review skipped: hotelReviewId=%d", hotelReviewID)
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

	// Update ratings summary
	var summary models.HotelRatingsSummary
	if err := db.First(&summary, "hotel_id = ?", hotel.ID).Error; err != nil {
		// New summary
		summary = models.HotelRatingsSummary{
			HotelID:       hotel.ID,
			TotalReviews:  1,
			TotalRating:   float64(review.Rating),
			AverageRating: float64(review.Rating),
			LastUpdated:   time.Now(),
		}
		db.Create(&summary)
	} else {
		// Update summary
		summary.TotalReviews += 1
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
