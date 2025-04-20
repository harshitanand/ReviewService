
![Go](https://img.shields.io/badge/Go-1.22-blue) ![Kafka](https://img.shields.io/badge/Kafka-Streaming-informational) ![Dockerized](https://img.shields.io/badge/Dockerized-yes-brightgreen) ![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg) ![Go CI](https://github.com/harshitanand/ReviewService/actions/workflows/go-ci.yml/badge.svg)


# 🚀 Review System Microservice

> **A high-performance, cloud-native review ingestion and aggregation system.**  
> Designed to handle **hotel reviews at scale per day** across multiple providers using a modern event-driven architecture with Go, Kafka, and PostgreSQL.

---

## ✨ What Makes This Awesome?

| 🧩 Feature | ⚙️ Description |
|-----------|----------------|
| **Real-time Streaming** | Streams `.jl` files from S3 to Kafka and into PostgreSQL via concurrent consumers |
| **Race-safe Inserts** | Uses manual conflict resolution logic to ensure atomic writes |
| **Pre-aggregated Ratings** | Real-time update of hotel-level review summaries |
| **Idempotent Ingestion** | Automatically skips reprocessing already-handled files |
| **Scales Horizontally** | Kafka consumers can scale independently for massive throughput |
| **Mock Test Framework** | 15 days of auto-generated data with rating-aligned sentiments |
| **Cloud-native Design** | Fully containerized and ready for orchestration |

---

## 📦 Architecture Overview

```txt
          

        ┌────────────┐      Line-by-Line                 ┌────────────┐
        │   AWS S3   │    -------─────────────────────▶  │ Kafka Prod │
        │ (Daily .jl)│                                   └────┬───────┘
        └────┬───────┘                                        ▼
             │                                           ┌────────────┐
             ▼                                           │ Kafka Topic│
        ┌────────────┐                                   │ reviews.raw│
        │ Wait Logic │                                   └────┬───────┘
        └────────────┘                                        ▼
                                                    ┌───────────────┐
                                                    │ Kafka Consumer│ ← Worker Pool (8x)
                                                    └────-┬─────────┘
                                                          ▼
                                                ┌─────────────────────────────┐
                                                │PostgreSQL (Relational, GORM)│
                                                └─────────────────────────────┘
                                                              ▼
                                              ┌──────────────────────────────┐
                                              │ REST API for Hotel + Ratings │
                                              └──────────────────────────────┘

``` 



---

## ⚙️ Tech Stack

| Layer            | Technology           |
|------------------|----------------------|
| Language         | Go (v1.22+)           |
| Framework        | Echo for REST APIs   |
| DB               | PostgreSQL + GORM    |
| Queue            | Kafka (segmentio/kafka-go) |
| Object Storage   | AWS S3 (Go SDK v2)   |
| Packaging        | Docker + Docker Compose |
| Docs             | Swagger via swaggo   |
| Testing          | Mock .jl generator   |

---

---

## 📌 Assumptions

- Incoming `.jl` files from S3 are line-delimited JSON and well-structured.
- The S3 bucket is accessible with provided credentials and region.
- Kafka brokers and PostgreSQL are reachable and ready at container start time.
- Review data includes consistent fields like `hotelId`, `platform`, `comment.rating`, and `reviewerInfo`.

---

## 🔄 How Ingestion Works

The system automatically ingests review data daily via the following mechanism:

### ✅ Automated Daily Flow

1. **S3 File Detection**  
   On container start, the Go app looks for a `.jl` file named with today’s date (e.g., `2025-04-20.jl`) in the S3 bucket and path.

2. **Kafka Producer**  
   Reads the `.jl` file line by line and sends reviews as individual messages to a Kafka topic (`reviews.raw`).

3. **Kafka Consumer**  
   A concurrent consumer group ingests these messages using worker goroutines and processes them into PostgreSQL.

4. **Deduplication & Safety**  
   Duplicate reviews (based on `hotel_review_id`) are ignored. Hotel creation is race-safe and atomic.

5. **Rating Summary**  
   A `hotel_ratings_summary` table is automatically updated per review for fast read APIs.

---

## 🧪 How to Manually Trigger Ingestion (Optional)

If needed, place mock files into the `testdata/` folder and modify `main.go` to:

```go
handlers.IngestJLFileAsync("testdata/2025-04-21.jl")
```

---

## 🌐 Multi-Provider Support

The system is designed to support **reviews from multiple platforms** such as:

- ✅ Agoda
- ✅ Booking.com
- ✅ Expedia
- ✅ Any future providers...

### How It Works:

| Component        | Description |
|------------------|-------------|
| `platform` table | Tracks each review’s source platform (e.g., Agoda, Booking) |
| `Review` model   | Stores `PlatformID` as a foreign key |
| `Kafka Payload`  | Includes `platform` name from the `.jl` file |
| S3 Integration   | Review files can be uploaded to the same S3 prefix with mixed or split platform data |
| API              | Aggregates all reviews regardless of platform or filters by provider if needed |

---

## 🏗️ Project Structure

```bash
.
├── Dockerfile
├── README.md
├── docker-compose.yml
├── .env
├── .env.docker
├── wait-for-it.sh
├── go.mod
├── go.sum
├── internal/
│   └── ingestion/
│       ├── consumer.go
│       ├── producer.go
│       ├── processor.go
├── handlers/
│   ├── ingest.go
│   └── review.go
├── models/
│   └── models.go
├── routes/
│   └── routes.go
├── testdata/
│   ├── 2025-04-19.jl
│   ├── 2025-04-20.jl
│   ├── ...
└── scripts/
    └── generate_mock_reviews.py
```

---

## 🛠 Setup (Quick Start)

### 1️⃣ Clone and Configure

```bash
git clone https://github.com/harshitanand/ReviewService.git
cd ReviewService
```

### 2️⃣ Build and Run

```bash
docker-compose up --build
```

This will spin up:
- ✅ PostgreSQL
- ✅ Multi-broker Kafka cluster
- ✅ Zookeeper
- ✅ Go app with pre-ingestion
- ✅ Swagger UI at [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)

---

## 🔥 Endpoints

### `GET /hotels/{hotel_id}/reviews example: /hotels/4/reviews?page=1&limit=5&sort=rating_desc`

> Returns hotel’s average rating and recent 50 reviews.

```json
{
  "hotel": {
    "hotel_id": 123,
    "hotel_name": "Ocean Breeze Resort",
    "average_rating": 8.6,
    "review_count": 2031
  },
  "reviews": [
    {
      "rating": 9,
      "review_title": "Amazing stay!",
      "review_text": "Clean, quiet, perfect location.",
      "review_date": "2025-04-20",
      "country_name": "Canada",
      "review_group_name": "Family",
      "room_type_name": "Deluxe King Room"
    }
  ]
}
```

---

## 🧪 Mock Review Dataset

We’ve generated **1000 reviews/day** × **15 days** with:

- 🏨 25+ unique hotels
- 🛏️ 10 room types
- 🌍 Global reviewers
- 🎯 Ratings from 1 to 10 aligned with sentiment

> Drop files in `testdata/` or push them to your S3 bucket — ingestion is automatic and idempotent.

---

## ⚡ Engineering Highlights

| Area | Enhancement |
|------|-------------|
| 💾 **Atomic Inserts** | Avoided `FirstOrCreate` in favor of `SELECT` → `INSERT` → `fallback SELECT` to handle race conditions |
| 📚 **Normalized Schema** | Hotels, Platforms, Reviewers, and Reviews in fully normalized structure |
| 🧮 **Real-Time Summary** | Ratings summary table is updated per review insert to power analytics |
| 📁 **Per-day S3 ingestion** | Streams daily file once using UTC timestamps|
| 🧵 **Goroutine Worker Pool** | Kafka messages processed in parallel using a buffered channel and 8+ workers |
| 🪵 **Safe Panic Recovery** | Full recover-wrapped ingestion to ensure no ingestion failures kill the consumer |
| 🧵 **Concurrency** | Worker-pool based Kafka consumer with panic recovery |
| 🔐 **DB Safety** | Atomic insert logic for Hotel, Platform, Reviewer with race-safe retries |
| 🔄 **Deduplication** | Composite keys and unique constraints at the DB level |
| 📦 **Bulk Handling** | S3 ingestion uses batched Kafka producer for high throughput |
| 🧠 **Precomputed Metrics** | Average rating and total reviews updated in real time during ingestion |
| 🔧 **Configurable** | All runtime configs (Kafka, S3, DB) are .env driven |
| 🔁 **Idempotent File Reads** | Each S3 file is marked as processed to prevent re-ingestion |
| 📦 **Kafka Batching** | Bulk writes to Kafka for better producer throughput |
| 🌍 **.env Driven** | All runtime settings are configurable from environment variables |

---

## 🧠 Scaling Strategy

- Horizontally scale consumers using `docker-compose scale` or Kubernetes
- Kafka handles backpressure & buffering under load
- PostgreSQL summary table avoids repeated heavy aggregates
- Add Prometheus + Grafana for lag monitoring

---

## 📜 License

MIT © 2025 [Harshit Anand](https://github.com/harshitanand)
