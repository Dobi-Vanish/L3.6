.PHONY: build run docker-up docker-down

build:
	go build -o bin/server ./cmd/server

run:
	go run ./cmd/server

docker-build:
	docker-compose -f deployments/docker-compose.yml build --no-cache

docker-up:
	docker-compose -f deployments/docker-compose.yml up -d

docker-down:
	docker-compose -f deployments/docker-compose.yml down -v

docker-all:
	docker-compose -f deployments/docker-compose.yml build --no-cache
	docker-compose -f deployments/docker-compose.yml up -d