include .env.example
-include .env
export
export GOOSE_DBSTRING := $(DATABASE_URL)

GOLANGCI_LINT ?= $(shell go env GOPATH)/bin/golangci-lint

.PHONY: run generate generate-mocks test test-integration lint migrate migrate-up migrate-down migrate-status require-env
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

generate-mocks:
	go generate ./internal/usecase/trip/... ./internal/http/trip/...

test:
	go test -race ./...
	@coverage_file=$$(mktemp); \
		trap 'rm -f "$$coverage_file"' EXIT; \
		go test -race -coverprofile="$$coverage_file" ./internal/domain/... ./internal/usecase/... && \
		go tool cover -func="$$coverage_file" | awk '/^total:/ { print "domain + usecase total coverage:", $$NF }'

test-integration: require-env migrate-up
	go test -race -tags=integration -count=2 ./internal/repository/trip/...

lint:
	$(GOLANGCI_LINT) run ./...

migrate-up: require-env
	go tool goose -dir migrations up

migrate-down: require-env
	go tool goose -dir migrations down

migrate-status: require-env
	go tool goose -dir migrations status
