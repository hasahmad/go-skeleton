include .envrc

SMTP_PORT ?= 25
TRUSTED_ORIGINS ?= localhost 127.0.0.1

# ==================================================================================== #
# HELPERS
# ==================================================================================== #

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

.PHONY: confirm
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## run/api: run the cmd/api application
.PHONY: run/api
run/api:
	go run ./cmd/api/main.go -db-dsn=${DB_DSN} -smtp-port=${SMTP_PORT} -cors-trusted-origins=${TRUSTED_ORIGINS}

## db/psql: connect to the database using psql
.PHONY: db/psql
db/psql:
	psql ${DB_DSN}

## db/migrations/new name=$1: create a new database migration
.PHONY: db/migrations/new
db/migrations/new:
	@echo 'Creating migration files for ${name}...'
	goose -dir=./migrations create ${name} go

## db/migrations/up: apply all up database migrations
.PHONY: db/migrations/up
db/migrations/up: build/migrate confirm
	@echo 'Running up migrations...'
	DB_DSN=${DB_DSN} ./bin/migrate -dir ./migrations up

## db/migrations/down: revert last database migration
.PHONY: db/migrations/down
db/migrations/down: build/migrate confirm
	@echo 'Reverting last migration...'
	DB_DSN=${DB_DSN} ./bin/migrate -dir ./migrations down

## db/migrations/status: show database migrations status
.PHONY: db/migrations/status
db/migrations/status: build/migrate
	DB_DSN=${DB_DSN} ./bin/migrate -dir migrations status

# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## audit: tidy dependencies and format, vet and test all code
.PHONY: audit
audit:
	@echo 'Tidying and verifying module dependencies...'
	go mod tidy
	go mod verify
	@echo 'Formatting code...'
	go fmt ./...
	@echo 'Vetting code...'
	go vet ./...
	staticcheck ./...
	@echo 'Running tests...'
	go test -race -vet=off ./...

## vendor: tidy and vendor dependencies
.PHONY: vendor
vendor:
	@echo 'Tidying and verifying module dependencies...'
	go mod tidy
	go mod verify
	@echo 'Vendoring dependencies...'
	go mod vendor

# ==================================================================================== #
# BUILD
# ==================================================================================== #

current_time = $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
git_description = $(shell git describe --always --dirty --tags --long)
linker_flags = '-s -X main.buildTime=${current_time} -X main.version=${git_description}'

## build/api: build the cmd/api application
.PHONY: build/api
build/api:
	@echo 'Building cmd/api...'
	go build -ldflags=${linker_flags} -o=./bin/api ./cmd/api
	GOOS=linux GOARCH=amd64 go build -ldflags=${linker_flags} -o=./bin/linux_amd64/api ./cmd/api

## build/migrate: build the cmd/migrate application
.PHONY: build/migrate
build/migrate:
	go build -o=./bin/migrate ./cmd/migrate
	GOOS=linux GOARCH=amd64 go build -o=./bin/linux_amd64/migrate ./cmd/migrate
