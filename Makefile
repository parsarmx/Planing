.PHONY: setup build build-api run dev test docker db migrate sqlc
SQLC ?= sqlc
setup:
	cd frontend && npm ci
build: build-api
	cd frontend && npm run build
build-api:
	cd backend && go build -o bin/poker .
run: build
	cd backend && ./bin/poker serve
db:
	./scripts/postgres.sh start
migrate: build-api
	cd backend && ./bin/poker pg_create && ./bin/poker pg_migrate
sqlc:
	cd backend && $(SQLC) generate
dev:
	./scripts/dev.sh
test:
	cd backend && go test -race ./... && go vet ./...
	cd frontend && npm test && npm run build
docker:
	docker compose up --build
