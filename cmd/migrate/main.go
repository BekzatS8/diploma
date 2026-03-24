package main

import (
	"flag"
	"log"

	"buhpro/internal/platform/db"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	direction := flag.String("direction", "up", "migration direction: up|down")
	databaseURL := flag.String("database-url", "", "postgres connection string")
	migrationsPath := flag.String("migrations-path", "./migrations", "path to migrations directory")
	flag.Parse()

	if *databaseURL == "" {
		log.Fatal("database-url is required")
	}

	switch *direction {
	case "up":
		if err := db.ApplyMigrations(*databaseURL, *migrationsPath); err != nil {
			log.Fatalf("apply migrations up: %v", err)
		}
		log.Println("migrations applied")
	case "down":
		if err := db.RollbackOne(*databaseURL, *migrationsPath); err != nil {
			log.Fatalf("apply migrations down: %v", err)
		}
		log.Println("migration rollback complete")
	default:
		log.Fatalf("unknown direction: %s", *direction)
	}
}
