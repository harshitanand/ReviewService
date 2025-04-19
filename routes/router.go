package routes

import (
	"review-system/handlers"

	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"
)

func SetupRoutesWith(e *echo.Echo) {
	e.GET("/hotels/:id/reviews", handlers.GetHotelReviews)
	e.GET("/swagger/*", echoSwagger.WrapHandler)
	e.Logger.Fatal(e.Start(":8080"))
}
