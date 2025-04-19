package main

import (
    "fmt"
    "os"
    "review-system/handlers"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Usage: go run ingest_s3.go <filename.jl>")
        return
    }
    filename := os.Args[1]
    err := handlers.IngestJLFile(filename)
    if err != nil {
        fmt.Println("Ingestion failed:", err)
        return
    }
    fmt.Println("Ingestion successful for:", filename)
}