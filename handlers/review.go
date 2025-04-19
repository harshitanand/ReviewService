package handlers

import (
	"net/http"
	"review-system/models"

	"github.com/labstack/echo/v4"
)

// GetHotelReviews godoc
// @Summary Get hotel reviews and overall rating
// @Description Returns average rating and recent reviews for a hotel
// @Tags reviews
// @Produce json
// @Param hotel_id path int true "Hotel ID"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} echo.Map
// @Router /hotels/{hotel_id}/reviews [get]
func GetHotelReviews(c echo.Context) error {
	hotelID := c.Param("id")
	if hotelID == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Missing hotel_id in URL"})
	}
	db := models.GetDB()

	var summary models.AggregatedHotelReview
	err := db.Raw(`
        SELECT h.id as hotel_id, h.name as hotel_name, ROUND(AVG(r.rating)::numeric, 2) AS average_rating, COUNT(*) as review_count
        FROM reviews r
        JOIN hotels h ON h.id = r.hotel_id
        WHERE r.hotel_id = ?
        GROUP BY h.id
    `, hotelID).Scan(&summary).Error

	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to fetch summary"})
	}

	var reviews []map[string]interface{}
	err = db.Raw(`
        SELECT r.rating, r.review_title, r.review_text, r.review_date, rv.country_name, rv.review_group_name, rv.room_type_name
        FROM reviews r
        JOIN reviewers rv ON rv.id = r.reviewer_id
        WHERE r.hotel_id = ?
        ORDER BY r.review_date DESC
        LIMIT 50
    `, hotelID).Scan(&reviews).Error

	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to fetch reviews"})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"hotel":   summary,
		"reviews": reviews,
	})
}
