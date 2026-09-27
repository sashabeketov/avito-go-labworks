include .env.example
include .env
export

.PHONY: run generate migrate-down migrate-status migrate
migrate: migrate-up

run:
	go run ./cmd/trip-service

generate:
	 mkdir -p internal/generated
	 go tool oapi-codegen \
     		-generate types,chi-server \
     		-package api \
     		-o internal/generated/api.gen.go \
     		contracts/openapi/trip-service.openapi.yaml

migrate-up:
	go tool goose -dir migrations postgres "$(DATABASE_URL)" up

migrate-down:
	go tool goose -dir migrations postgres "$(DATABASE_URL)" down

migrate-status:
	go tool goose -dir migrations postgres "$(DATABASE_URL)" status