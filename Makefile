DATABASE_URL ?= postgres://mellow:mellow@localhost:5432/mellow?sslmode=disable
MIGRATIONS   := internal/db/migrations

.PHONY: build run test vet tidy sqlc migrate-up migrate-down migrate-force pg-up pg-down

build:
	go build ./...

run:
	go run ./cmd/mellow

test:
	go test ./...

vet:
	go vet ./...

tidy:
	go mod tidy

sqlc:
	sqlc generate

migrate-up:
	migrate -path $(MIGRATIONS) -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path $(MIGRATIONS) -database "$(DATABASE_URL)" down 1

migrate-force:
	migrate -path $(MIGRATIONS) -database "$(DATABASE_URL)" force $(V)

# Throwaway local Postgres for development.
pg-up:
	docker run -d --name mellow-pg -p 5432:5432 \
		-e POSTGRES_USER=mellow -e POSTGRES_PASSWORD=mellow -e POSTGRES_DB=mellow \
		postgres:16

pg-down:
	docker rm -f mellow-pg
