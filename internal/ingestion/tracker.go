package ingestion

import (
	"bufio"
	"log"
	"os"
)

func alreadyProcessed(filename string) bool {
	file, err := os.Open(ProcessedMarker)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if scanner.Text() == filename {
			return true
		}
	}
	return false
}

func markAsProcessed(filename string) {
	f, err := os.OpenFile(ProcessedMarker, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("⚠️  Couldn't mark processed: %v", err)
		return
	}
	defer f.Close()
	f.WriteString(filename + "\n")
}
