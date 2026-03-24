APP_NAME ?= buhpro-api
BINARY_PATH ?= bin/$(APP_NAME)
DB_URL ?= postgres://buhpro:buhpro@localhost:5432/buhpro?sslmode=disable
MIGRATIONS_DIR ?= ./migrations
COMPOSE_FILE ?= docker-compose.yml

.PHONY: run test build fmt tidy migrate-up migrate-down compose-up compose-down bootstrap-admin seed-demo

run:
	go run ./cmd/api

test:
	go test ./...

build:
	go build -o $(BINARY_PATH) ./cmd/api

fmt:
	gofmt -w $(shell find cmd internal -name '*.go')

tidy:
	go mod tidy

migrate-up:
	go run ./cmd/migrate -direction up -database-url '$(DB_URL)' -migrations-path '$(MIGRATIONS_DIR)'

migrate-down:
	go run ./cmd/migrate -direction down -database-url '$(DB_URL)' -migrations-path '$(MIGRATIONS_DIR)'

compose-up:
	docker compose -f $(COMPOSE_FILE) up --build -d

compose-down:
	docker compose -f $(COMPOSE_FILE) down --remove-orphans

bootstrap-admin:
	go run ./cmd/bootstrap-admin

seed-demo:
	go run ./cmd/seed-demo
