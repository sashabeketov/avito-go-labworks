include .env.example
-include .env
export
export GOOSE_DBSTRING := $(DATABASE_URL)

.PHONY: run generate test migrate migrate-up migrate-down migrate-status require-env
migrate: migrate-up

require-env:
	@test -f .env || { echo "Run tripgoctl connect to create .env" >&2; exit 1; }

run: require-env
	go run ./cmd/trip-service

generate:
	mkdir -p internal/generated
	go tool oapi-codegen \
		-generate types,chi-server \
		-package api \
		-o internal/generated/api.gen.go \
		contracts/openapi/trip-service.openapi.yaml

test:
	go test -race ./...

migrate-up: require-env
	go tool goose -dir migrations up

migrate-down: require-env
	go tool goose -dir migrations down

migrate-status: require-env
	go tool goose -dir migrations status
