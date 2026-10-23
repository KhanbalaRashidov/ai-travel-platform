.PHONY: run-api build-api tidy fmt vet up down logs migrate-up migrate-down

DB_URL ?= postgres://postgres:postgres@localhost:5432/ai_travel?sslmode=disable
COMPOSE := docker compose -f deploy/docker-compose.yml

run-api:
	go run ./services/gateway

build-api:
	go build -trimpath -o bin/gateway ./services/gateway

tidy:
	go mod tidy

fmt:
	gofmt -w .

vet:
	go vet ./...

up:
	$(COMPOSE) up -d

down:
	$(COMPOSE) down

logs:
	$(COMPOSE) logs -f

# Apply migrations to the LOCAL dev DB only (ephemeral). Requires psql.
migrate-up:
	psql "$(DB_URL)" -f db/migrations/0001_init.up.sql

migrate-down:
	psql "$(DB_URL)" -f db/migrations/0001_init.down.sql
