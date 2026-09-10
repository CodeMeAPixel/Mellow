ifneq (,$(wildcard .env))
include .env
export
endif

MIGRATIONS := internal/db/migrations

.PHONY: build run test vet tidy sqlc migrate migrate-up migrate-down migrate-force pg-up pg-down

build:
	go build ./...

run:
	go run ./cmd/mellow

migrate:
	MELLOW_MIGRATE_ONLY=1 go run ./cmd/mellow

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

pg-up:
	docker run -d --name $(PG_CONTAINER) -p $(PG_PORT):5432 \
		-e POSTGRES_USER=$(PG_USER) -e POSTGRES_PASSWORD=$(PG_PASSWORD) -e POSTGRES_DB=$(PG_DB) \
		$(PG_IMAGE)

pg-down:
	docker rm -f $(PG_CONTAINER)
