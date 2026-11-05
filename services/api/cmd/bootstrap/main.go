// Command bootstrap applies migrations and loads the deterministic demo seed.
package main

import (
	"context"
	"log"
	"os"

	"github.com/grupon/grupon/services/api/internal/seed"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	if err := seed.Run(context.Background(), databaseURL); err != nil {
		log.Fatal(err)
	}
	log.Println("demo database bootstrapped")
}
