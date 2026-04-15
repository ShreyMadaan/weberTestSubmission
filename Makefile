APP_NAME=auctioncore
DB_URL=postgres://auctioncore:auctioncore@localhost:5432/auctioncore?sslmode=disable

.PHONY: up down run tidy fmt

up:
docker compose up -d

down:
docker compose down

run:
go run ./cmd/api

tidy:
go mod tidy

fmt:
go fmt ./...
migrate:
go run ./cmd/migrate

seed:
go run ./scripts/seed.go
