package ingestion

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
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
	KafkaBrokers    = strings.Split(getEnv("KAFKA_BROKERS", ""), ",")
	S3Bucket        = getEnv("S3_BUCKET", "your-s3-bucket")
	S3Prefix        = getEnv("S3_PREFIX", "your/s3/path")
	ProcessedMarker = getEnv("PROCESSED_LOG", "processed.log")
	envMap          = make(map[string]string)
	BatchSize       = 50
)

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func EnsureTopic() {
	conn, err := kafka.Dial("tcp", "kafka1:29092")
	if err != nil {
		log.Printf("⚠️ Kafka dial error: %v", err)
		return
	}
	defer conn.Close()

	topic := os.Getenv("KAFKA_TOPIC")
	if topic == "" {
		topic = "reviews.raw"
	}

	err = conn.CreateTopics(kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     3,
		ReplicationFactor: 2,
	})
	if err != nil {
		log.Printf("⚠️ Topic creation failed (might already exist): %v", err)
	} else {
		log.Printf("✅ Kafka topic %s ensured", topic)
	}
}

func StartS3StreamIngestion() {
	EnsureTopic()

	envMap := make(map[string]string)
	for _, e := range os.Environ() {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	go func() {
		today := time.Now().UTC().Format("2006-01-02")
		key := fmt.Sprintf("%s/%s.jl", envMap["S3_PREFIX"], today)
		marker := fmt.Sprintf("%s/%s", envMap["S3_BUCKET"], key)

		if alreadyProcessed(marker) {
			log.Printf("✅ File already processed: %s", marker)
			return
		}

		if err := StreamAndBulkProduceFromS3Read(envMap["S3_BUCKET"], key); err != nil {
			log.Printf("❌ Error streaming from S3: %v", err)
		} else {
			markAsProcessed(marker)
			log.Printf("✅ Successfully streamed: %s", marker)
		}
	}()
}

func StreamAndBulkProduceFromS3Read(bucket, key string) error {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "ap-south-1" // fallback
	}

	// Construct public URL
	url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucket, region, key)
	log.Printf("🌐 Fetching public S3 file: %s", url)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch public S3 file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("non-200 status from S3: %s", resp.Status)
	}

	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  KafkaBrokers,
		Topic:    KafkaTopic,
		Balancer: &kafka.LeastBytes{},
	})
	defer writer.Close()

	log.Printf("📤 Streaming public S3 and producing in batches of %d...", BatchSize)

	scanner := bufio.NewScanner(resp.Body)
	var batch []kafka.Message

	for scanner.Scan() {
		line := scanner.Text()
		batch = append(batch, kafka.Message{Value: []byte(line)})

		if len(batch) >= BatchSize {
			if err := writer.WriteMessages(context.Background(), batch...); err != nil {
				return fmt.Errorf("❌ failed to write batch: %w", err)
			}
			batch = batch[:0]
		}
	}

	if len(batch) > 0 {
		if err := writer.WriteMessages(context.Background(), batch...); err != nil {
			return fmt.Errorf("❌ failed to write final batch: %w", err)
		}
	}

	return scanner.Err()
}

func StreamAndBulkProduceFromS3(bucket, key string) error {
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

	log.Printf("📤 Producing records in batches of %d...", BatchSize)

	scanner := bufio.NewScanner(resp.Body)
	var batch []kafka.Message

	for scanner.Scan() {
		line := scanner.Text()
		batch = append(batch, kafka.Message{Value: []byte(line)})

		if len(batch) >= BatchSize {
			if err := writer.WriteMessages(context.Background(), batch...); err != nil {
				return fmt.Errorf("❌ failed to write batch: %w", err)
			}
			batch = batch[:0]
		}
	}

	if len(batch) > 0 {
		if err := writer.WriteMessages(context.Background(), batch...); err != nil {
			return fmt.Errorf("❌ failed to write final batch: %w", err)
		}
	}

	return scanner.Err()
}
