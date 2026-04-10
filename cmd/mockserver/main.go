package main

import (
	"log"

	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/mockserver"
)

func main() {
	cfg := mockserver.LoadConfig()
	if err := mockserver.Run(cfg); err != nil {
		log.Fatalf("mock server error: %v", err)
	}
}
