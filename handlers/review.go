package handlers

import (
	"net/http"
	"strconv"

	"review-system/models"

	"github.com/labstack/echo/v4"
)

// GetHotelReviews godoc
// @Summary Get hotel reviews and overall rating
// @Description Returns average rating and paginated reviews for a hotel
// @Tags reviews
// @Produce json
// @Param hotel_id path int true "Hotel ID"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Reviews per page" default(20)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} echo.Map
// @Failure 200 {object} echo.Map
// @Router /hotels/{hotel_id}/reviews [get]
func GetHotelReviews(c echo.Context) error {
	hotelID := c.Param("id")
	if hotelID == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Missing hotel_id in URL"})
	}

	// Defaults
	page := 1
	limit := 20

	// Parse & validate query params
	if p := c.QueryParam("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if l := c.QueryParam("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	offset := (page - 1) * limit
	db := models.GetDB()

	// Fetch summary
	var summary models.AggregatedHotelReview
	if err := db.Raw(`
        SELECT h.id as hotel_id, h.name as hotel_name, ROUND(AVG(r.rating)::numeric, 2) AS average_rating, COUNT(*) as review_count
        FROM reviews r
        JOIN hotels h ON h.id = r.hotel_id
        WHERE r.hotel_id = ?
        GROUP BY h.id
    `, hotelID).Scan(&summary).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to fetch summary"})
	}

	// Fetch reviews paginated
	var reviews []map[string]interface{}
	if err := db.Raw(`
        SELECT r.rating, r.review_title, r.review_text, r.review_date, rv.country_name, rv.review_group_name, rv.room_type_name
        FROM reviews r
        JOIN reviewers rv ON rv.id = r.reviewer_id
        WHERE r.hotel_id = ?
        ORDER BY r.review_date DESC
        LIMIT ? OFFSET ?
    `, hotelID, limit, offset).Scan(&reviews).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to fetch reviews"})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"hotel": summary,
		"pagination": echo.Map{
			"page":  page,
			"limit": limit,
		},
		"reviews": reviews,
	})
}
