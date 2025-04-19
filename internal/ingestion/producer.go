package ingestion

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/segmentio/kafka-go"
)

var (
	KafkaTopic      = getEnv("KAFKA_TOPIC", "reviews.raw")
	KafkaBrokers    = strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ",")
	S3Bucket        = getEnv("S3_BUCKET", "zuzu-3p-reviews")
	S3Prefix        = getEnv("S3_PREFIX", "reviews-dump/jl")
	ProcessedMarker = getEnv("PROCESSED_LOG", "processed.log")
)

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func StartS3StreamIngestion() {
	go func() {
		today := time.Now().UTC().Format("2006-01-02")
		key := fmt.Sprintf("%s/%s.jl", S3Prefix, today)
		marker := fmt.Sprintf("%s/%s", S3Bucket, key)

		if alreadyProcessed(marker) {
			log.Printf("✅ File already processed: %s", marker)
			return
		}

		if err := streamAndProduceFileFromS3(S3Bucket, key); err != nil {
			log.Printf("❌ Error streaming from S3: %v", err)
		} else {
			markAsProcessed(marker)
			log.Printf("✅ Successfully streamed: %s", marker)
		}
	}()
}

func streamAndProduceFileFromS3(bucket, key string) error {
	awsKey := os.Getenv("AWS_ACCESS_KEY_ID")
	awsSecret := os.Getenv("AWS_SECRET_ACCESS_KEY")
	awsRegion := os.Getenv("AWS_REGION")

	if awsKey == "" || awsSecret == "" || awsRegion == "" {
		return fmt.Errorf("missing AWS credentials or region in environment")
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(awsKey, awsSecret, "")),
		config.WithRegion(awsRegion),
	)
	if err != nil {
		return fmt.Errorf("unable to load AWS config: %w", err)
	}

	s3client := s3.NewFromConfig(cfg)

	resp, err := s3client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to fetch S3 object: %s %s %w", bucket, key, err)
	}
	defer resp.Body.Close()

	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  KafkaBrokers,
		Topic:    KafkaTopic,
		Balancer: &kafka.LeastBytes{},
	})
	defer writer.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		err := writer.WriteMessages(context.Background(), kafka.Message{Value: []byte(line)})
		if err != nil {
			return err
		}
	}

	return scanner.Err()
}
