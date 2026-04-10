package main

import (
	"log"

	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/config"
	"github.com/bryanperez/laguna-escondida-marketing/backend/internal/database"
	"github.com/bryanperez/laguna-escondida-marketing/backend/migrations"
	"gorm.io/gorm/logger"
)

func main() {
	log.Println("Running database seeds...")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	db, err := database.Connect(cfg.GetDSN(), logger.Warn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	database.SeedFS = migrations.SeedFiles
	database.SeedDir = migrations.SeedDir

	if err := database.RunSeeds(db); err != nil {
		log.Fatalf("Failed to run seeds: %v", err)
	}

	log.Println("✓ Done")
}
