package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	ratingsmodule "buhpro/internal/modules/ratingsanctions"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	databaseURL := flag.String("database-url", "", "postgres connection string; defaults to DB_URL")
	timeout := flag.Duration("timeout", 30*time.Second, "command timeout")
	flag.Parse()

	dbURL := *databaseURL
	if dbURL == "" {
		dbURL = os.Getenv("DB_URL")
	}
	if dbURL == "" {
		log.Fatal("database-url or DB_URL is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("init db: %v", err)
	}
	defer pool.Close()

	repo := ratingsmodule.NewRepository(pool, ratingsmodule.RepositoryOptions{})
	result, err := repo.ExpireDue(ctx, "", "command")
	if err != nil {
		log.Fatalf("expire sanctions: %v", err)
	}

	log.Printf("expired sanctions: %d", result.ExpiredCount)
}
