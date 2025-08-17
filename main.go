package main

import (
	"log"

	"github.com/joho/godotenv"
	"ingest-edge/internal/httpserver"
)

func main() {
	_ = godotenv.Load()
	if err := httpserver.Start(); err != nil {
		log.Fatal(err)
	}
}
