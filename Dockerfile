# Stage 1: Build
FROM golang:1.23.8-alpine AS builder

WORKDIR /app

# Leverage caching for module downloads
COPY go.mod go.sum ./
RUN go mod download

# Copy all source files
COPY . .

# Copy wait-for-it script and make it executable
COPY wait-for-it.sh /wait-for-it.sh
RUN chmod +x /wait-for-it.sh

# Build Go binary
RUN go build -o app .

# Stage 2: Runtime with shell and netcat support
FROM alpine:3.18

WORKDIR /app

# Install netcat (required by wait-for-it)
RUN apk add --no-cache netcat-openbsd

# Copy built binary
COPY --from=builder /app/app .

# Copy wait-for-it script
COPY --from=builder /wait-for-it.sh /wait-for-it.sh
RUN chmod +x /wait-for-it.sh

# Optionally copy .env.docker (rename it here)
COPY .env.docker .env

# Set final entrypoint to wait and run app
ENTRYPOINT ["/wait-for-it.sh", "postgres", "5432", "--", "/wait-for-it.sh", "kafka1", "29092", "--", "/wait-for-it.sh", "kafka2", "29093", "--", "/app/app"]
