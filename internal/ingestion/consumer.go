package ingestion

import (
	"context"
	"encoding/json"
	"log"

	"review-system/models"

	"github.com/segmentio/kafka-go"
)

func StartKafkaConsumer() {
	go func() {
		r := kafka.NewReader(kafka.ReaderConfig{
			Brokers:  KafkaBrokers,
			Topic:    KafkaTopic,
			GroupID:  getEnv("KAFKA_CONSUMER_GROUP", "review-ingestors"),
			MinBytes: 10e3,
			MaxBytes: 10e6,
		})
		defer r.Close()

		log.Println("🚀 Kafka consumer started")
		for {
			m, err := r.ReadMessage(context.Background())
			if err != nil {
				log.Printf("⚠️  Kafka read error: %v", err)
				continue
			}
			var raw map[string]interface{}
			if err := json.Unmarshal(m.Value, &raw); err != nil {
				log.Printf("⚠️  Invalid JSON from Kafka: %v", err)
				continue
			}
			ProcessJLLine(raw, models.GetDB())
		}
	}()
}
