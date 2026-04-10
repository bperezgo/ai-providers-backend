package database

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"sort"
	"strings"

	"gorm.io/gorm"
)

// MigrationFS holds the embedded SQL migration files.
// Set from main.go or wherever the embed directive lives.
var MigrationFS embed.FS

// MigrationDir is the directory inside MigrationFS that contains .sql files.
var MigrationDir = "migrations/sql"

// RunMigrations executes all SQL migration files in order.
// Each migration uses IF NOT EXISTS so it is safe to run repeatedly.
func RunMigrations(db *gorm.DB) error {
	log.Println("Running database migrations...")

	entries, err := fs.ReadDir(MigrationFS, MigrationDir)
	if err != nil {
		return fmt.Errorf("failed to read migration files: %w", err)
	}

	// Sort by filename to guarantee execution order (001_, 002_, ...)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		path := MigrationDir + "/" + entry.Name()
		content, err := fs.ReadFile(MigrationFS, path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", entry.Name(), err)
		}

		log.Printf("  Running %s ...", entry.Name())
		if err := db.Exec(string(content)).Error; err != nil {
			return fmt.Errorf("migration %s failed: %w", entry.Name(), err)
		}
	}

	log.Println("✓ Database migrations completed successfully")
	return nil
}

// SeedFS holds the embedded SQL seed files.
// Set from main.go or wherever the embed directive lives.
var SeedFS embed.FS

// SeedDir is the directory inside SeedFS that contains seed .sql files.
var SeedDir = "seeds"

// RunSeeds executes all SQL seed files in order.
// Uses ON CONFLICT / upsert so it is safe to run repeatedly.
func RunSeeds(db *gorm.DB) error {
	log.Println("Running database seeds...")

	entries, err := fs.ReadDir(SeedFS, SeedDir)
	if err != nil {
		return fmt.Errorf("failed to read seed files: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		path := SeedDir + "/" + entry.Name()
		content, err := fs.ReadFile(SeedFS, path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", entry.Name(), err)
		}

		log.Printf("  Seeding %s ...", entry.Name())
		if err := db.Exec(string(content)).Error; err != nil {
			return fmt.Errorf("seed %s failed: %w", entry.Name(), err)
		}
	}

	log.Println("✓ Database seeds completed successfully")
	return nil
}
