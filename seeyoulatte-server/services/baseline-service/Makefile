# baseline-service has no DB, so migrate-* targets are intentionally omitted.

# Load the .env file if it exists
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

.PHONY: build run dev test clean docker-up docker-down

# --- build / run ---

build:
	@go build -o bin/baseline-service cmd/server/main.go

run:
	@go run cmd/server/main.go

dev:
	@air

clean:
	@rm -rf bin/ tmp/

# --- tests ---

test:
	@go test -v ./... -cover

# --- docker (proxy to root docker-compose) ---

docker-up:
	@docker compose -f ../../docker-compose.yml up -d seeyoulatte-consul seeyoulatte-rabbitmq

docker-down:
	@docker compose -f ../../docker-compose.yml down
