.PHONY: run build test race vet web-install web-test web-build migrate compose-up compose-down

run:
	go run ./cmd/server

build:
	go build ./...

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

web-install:
	cd web && npm ci

web-test:
	cd web && npm test

web-build:
	cd web && npm run build

migrate:
	psql "$(DATABASE_URL)" -v ON_ERROR_STOP=1 -f migrations/001_init.sql -f migrations/002_seed.sql

compose-up:
	docker compose up --build

compose-down:
	docker compose down
