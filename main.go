package main

import (
	"log"
	"time"

	_ "review-system/docs"
	"review-system/internal/ingestion"
	"review-system/models"
	"review-system/routes"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"
)

// @title Hotel Review API
// @version 1.0
// @description API to fetch hotel reviews and ratings.
// @host localhost:8080
// @BasePath /

func main() {
	// Load .env file for local development
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  No .env file found — using system environment vars")
	}

	// Initialize DB using environment
	models.InitDB()
	// Check if reviews already exist
	var count int64
	models.GetDB().Model(&models.Review{}).Count(&count)

	// if count == 0 {
	// 	log.Println("📥 No reviews found. Ingesting test data from testdata/sample.jl...")
	// handlers.IngestJLFileAsync("testdata/sample_1000.jl") // ✅ fire and forget
	// }

	// Start Kafka consumer to ingest reviews
	ingestion.StartKafkaConsumer()

	// Periodic daily ingestion check
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	var lastIngested string

	go func() {
		for {
			now := time.Now().UTC()
			today := now.Format("2006-01-02")

			if today != lastIngested {
				log.Printf("📆 Running ingestion for %s\n", today)
				ingestion.StartS3StreamIngestion()
				lastIngested = today
			}

			<-ticker.C
		}
	}()

	// Start Echo server
	e := echo.New()
	routes.SetupRoutesWith(e)
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	log.Println("🚀 Server starting on http://localhost:8080 ...")
	e.Logger.Fatal(e.Start(":8080"))
}
