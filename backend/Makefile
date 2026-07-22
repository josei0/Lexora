DATABASE_URL ?= $(shell grep -E '^DATABASE_URL=' ../.env | cut -d= -f2-)

.PHONY: run gen migrate-up migrate-down seed test tidy tools

run:
	go run ./cmd/server

gen:
	oapi-codegen -config oapi-codegen.yaml ../openapi/openapi.yaml

migrate-up:
	migrate -path db/migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path db/migrations -database "$(DATABASE_URL)" down 1

seed:
	go run ./db/seed

test:
	go test ./...

tidy:
	go mod tidy

# install dev tools
tools:
	go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
