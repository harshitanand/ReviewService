package ingestion

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"review-system/models"

	"github.com/segmentio/kafka-go"
)

var (
	KafkaConsumerGroup = getEnv("KAFKA_CONSUMER_GROUP", "review-ingestors")
)

func StartKafkaConsumer() {
	go func() {
		concurrency := 8
		messageCh := make(chan kafka.Message, 1000)

		// Start concurrent workers
		for i := 0; i < concurrency; i++ {
			go func(workerID int) {
				for msg := range messageCh {
					var raw map[string]interface{}
					if err := json.Unmarshal(msg.Value, &raw); err != nil {
						log.Printf("⚠️  [Worker %d] Invalid JSON: %v", workerID, err)
						continue
					}
					func() {
						defer func() {
							if r := recover(); r != nil {
								log.Printf("⚠️  [Worker %d] Panic recovered: %v", workerID, r)
							}
						}()

						// Process with silent handling of "record not found"
						ProcessJLLineWithSuppressedErrors(raw, models.GetDB())
					}()
				}
			}(i)
		}

		// Kafka reader
		r := kafka.NewReader(kafka.ReaderConfig{
			Brokers:         KafkaBrokers,
			Topic:           KafkaTopic,
			GroupID:         KafkaConsumerGroup,
			MinBytes:        1e4,
			MaxBytes:        10e6,
			MaxWait:         100 * time.Millisecond,
			QueueCapacity:   1000,
			CommitInterval:  1 * time.Second,
			ReadLagInterval: -1,
		})
		defer r.Close()

		log.Println("🚀 Kafka consumer started with concurrency =", concurrency)

		for {
			m, err := r.ReadMessage(context.Background())
			if err != nil {
				log.Printf("⚠️  Kafka read error: %v", err)
				continue
			}
			messageCh <- m
		}
	}()
}
