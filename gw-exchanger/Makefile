.PHONY: build run test clean migrate-up migrate-down proto help

# Переменные
APP_NAME=gw-exchanger
BINARY_NAME=exchanger-service
DOCKER_IMAGE=$(APP_NAME):latest
MIGRATIONS_PATH=./migrations
CONFIG_FILE=config.env
DB_URL=postgresql://postgres:1234@localhost:5433/exchanger?sslmode=disable
PROTO_PATH=./proto-exchange

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

## clean: Очистить бинарники и кэш
clean:
	@echo "Cleaning..."
	rm -rf ./bin
	rm -f coverage.out coverage.html exchanger.log
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

## migrate-force: Принудительно установить версию миграции
migrate-force:
	@if [ -z "$(VERSION)" ]; then echo "Error: VERSION is required. Usage: make migrate-force VERSION=1"; exit 1; fi
	@echo "Forcing migration version $(VERSION)..."
	migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" force $(VERSION)

## migrate-create: Создать новую миграцию
migrate-create:
	@if [ -z "$(NAME)" ]; then echo "Error: NAME is required. Usage: make migrate-create NAME=add_rates_table"; exit 1; fi
	@echo "Creating migration $(NAME)..."
	migrate create -ext sql -dir $(MIGRATIONS_PATH) -seq $(NAME)

## proto: Сгенерировать protobuf код
proto:
	@echo "Generating protobuf code..."
	cd $(PROTO_PATH) && \
	protoc --go_out=. --go_opt=paths=source_relative \
	       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	       exchange.proto
	@echo "Protobuf generation complete"

## proto-clean: Удалить сгенерированный protobuf код
proto-clean:
	@echo "Cleaning protobuf generated files..."
	rm -f $(PROTO_PATH)/*.pb.go

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

## grpc-test: Тест gRPC endpoint через grpcurl
grpc-test:
	@echo "Testing gRPC endpoint..."
	grpcurl -plaintext localhost:50051 list
	grpcurl -plaintext localhost:50051 exchange.ExchangeService/GetExchangeRates

.DEFAULT_GOAL := help