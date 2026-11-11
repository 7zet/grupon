// Command migrate applies pending database migrations and nothing else. It never seeds data,
// so it is the only database step that runs in production.
package main

import (
	"context"
	"log"
	"os"

	"github.com/dealo/dealo/services/api/internal/seed"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	dir := os.Getenv("MIGRATIONS_DIR")
	if dir == "" {
		dir = "/app/migrations"
	}
	if err := seed.MigrateOnly(context.Background(), databaseURL, dir); err != nil {
		log.Fatal(err)
	}
	log.Println("migrations applied")
}
