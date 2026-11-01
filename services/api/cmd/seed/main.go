// Command seed loads the deterministic development dataset. It refuses to run in production.
package main

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/grupon/grupon/services/api/internal/seed"
)

const defaultDatabaseURL = "postgres://app:app_dev_password@localhost:5432/app?sslmode=disable"

func main() {
	if strings.EqualFold(os.Getenv("APP_ENV"), "production") {
		log.Fatal("seed is disabled in production")
	}
	databaseURL := envOr("DATABASE_URL", defaultDatabaseURL)
	if err := seed.Run(context.Background(), databaseURL); err != nil {
		log.Fatal(err)
	}
	log.Println("seed complete: 5 cities, 11 categories, 30 merchants, 100 deals, 400 reviews")
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
