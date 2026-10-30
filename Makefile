.PHONY: up down build logs ps migrate-up test test-integration tidy

up:
	docker compose up --build -d

down:
	docker compose down

build:
	go build ./...

logs:
	docker compose logs -f app

ps:
	docker compose ps

migrate-up:
	docker compose run --rm migrate

test:
	go test ./...

test-integration:
	go test -tags=integration ./...

tidy:
	go mod tidy
