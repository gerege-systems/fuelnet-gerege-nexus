.PHONY: dev-backend dev-frontend up down migrate seed test build

DATABASE_URL ?= postgres://postgres:postgrespassword@localhost:5432/platform_db?sslmode=disable

dev-backend:
	cd backend && go run ./cmd/api

dev-frontend:
	cd frontend && npm run dev

up:
	docker-compose up -d

down:
	docker-compose down -v

migrate:
	cd backend && DATABASE_URL="$(DATABASE_URL)" go run ./cmd/migrate up

seed:
	cd backend && DATABASE_URL="$(DATABASE_URL)" go run ./cmd/api

test:
	cd backend && go test ./...

build:
	cd backend && go build ./...
	cd frontend && npm run build
