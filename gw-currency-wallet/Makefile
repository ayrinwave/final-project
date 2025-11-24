.PHONY: build run test clean migrate-up migrate-down docker-build docker-run swagger help

# Переменные
APP_NAME=gw-currency-wallet
BINARY_NAME=wallet-service
DOCKER_IMAGE=$(APP_NAME):latest
MIGRATIONS_PATH=./migrations
CONFIG_FILE=config.env
DB_URL=postgresql://postgres:1234@localhost:5432/wallet?sslmode=disable

# Go параметры
GOOS ?= linux
GOARCH ?= amd64
GO_BUILD_FLAGS=-ldflags="-s -w"

## help: Показать help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## build: Собрать бинарник
build:
	@echo "Building $(BINARY_NAME)..."
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(GO_BUILD_FLAGS) -o ./bin/$(BINARY_NAME) ./cmd/main.go
	@echo "Build complete: ./bin/$(BINARY_NAME)"

## run: Запустить приложение локально
run:
	@echo "Running $(APP_NAME)..."
	go run ./cmd/main.go -c $(CONFIG_FILE)

## test: Запустить тесты
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## test-short: Быстрые тесты (без integration)
test-short:
	@echo "Running short tests..."
	go test -v -short ./...

## clean: Очистить бинарники и кэш
clean:
	@echo "Cleaning..."
	rm -rf ./bin
	rm -f coverage.out coverage.html wallet.log
	go clean -cache -testcache
	@echo "Clean complete"

## migrate-up: Применить миграции
migrate-up:
	@echo "Running migrations up..."
	migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" up

## migrate-down: Откатить миграции
migrate-down:
	@echo "Running migrations down..."
	migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" down

## migrate-force: Принудительно установить версию миграции (использование: make migrate-force VERSION=1)
migrate-force:
	@if [ -z "$(VERSION)" ]; then echo "Error: VERSION is required. Usage: make migrate-force VERSION=1"; exit 1; fi
	@echo "Forcing migration version $(VERSION)..."
	migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" force $(VERSION)

## migrate-create: Создать новую миграцию (использование: make migrate-create NAME=add_users_table)
migrate-create:
	@if [ -z "$(NAME)" ]; then echo "Error: NAME is required. Usage: make migrate-create NAME=add_users_table"; exit 1; fi
	@echo "Creating migration $(NAME)..."
	migrate create -ext sql -dir $(MIGRATIONS_PATH) -seq $(NAME)

## swagger: Сгенерировать Swagger документацию
swagger:
	@echo "Generating Swagger docs..."
	swag init -g ./cmd/main.go -o ./docs
	@echo "Swagger docs generated in ./docs"

## lint: Запустить линтер
lint:
	@echo "Running linter..."
	golangci-lint run ./...

## fmt: Форматировать код
fmt:
	@echo "Formatting code..."
	go fmt ./...
	goimports -w .

## deps: Установить зависимости
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy
	@echo "Dependencies installed"

## dev: Запустить в dev режиме с hot reload (требует air)
dev:
	@echo "Starting development server with hot reload..."
	air

.DEFAULT_GOAL := help