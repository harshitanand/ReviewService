FROM golang:1.20-alpine
WORKDIR /app
COPY . .
RUN go mod tidy && go install github.com/swaggo/swag/cmd/swag@latest && swag init && go build -o server
CMD ["./server"]