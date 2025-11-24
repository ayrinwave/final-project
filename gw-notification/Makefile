.PHONY: build run test clean kafka-test mongo-shell help

# Переменные
APP_NAME=gw-notification
BINARY_NAME=notification-service
DOCKER_IMAGE=$(APP_NAME):latest
CONFIG_FILE=config.env

# Go параметры
GOOS ?= linux
GOARCH ?= amd64
GO_BUILD_FLAGS=-ldflags="-s -w"

# Kafka & MongoDB параметры
KAFKA_BROKER=localhost:9092
KAFKA_TOPIC=large-transfers
MONGO_URI=mongodb://admin:admin123@localhost:27017

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
	rm -f coverage.out coverage.html notification.log
	go clean -cache -testcache
	@echo "Clean complete"

## kafka-test: Отправить тестовое сообщение в Kafka
kafka-test:
	@echo "Sending test message to Kafka..."
	@echo '{"transaction_id":"test-'$(shell date +%s)'","user_id":"550e8400-e29b-41d4-a716-446655440000","from_currency":"USD","to_currency":"EUR","amount":35000.0,"exchanged_amount":32200.0,"rate":0.92,"timestamp":"'$(shell date -u +%Y-%m-%dT%H:%M:%SZ)'"}' | \
	docker exec -i kafka kafka-console-producer --broker-list localhost:9092 --topic $(KAFKA_TOPIC)
	@echo "Test message sent"

## kafka-consume: Читать сообщения из Kafka (для отладки)
kafka-consume:
	@echo "Consuming messages from Kafka topic: $(KAFKA_TOPIC)"
	docker exec -it kafka kafka-console-consumer --bootstrap-server localhost:9092 --topic $(KAFKA_TOPIC) --from-beginning

## kafka-topics: Показать все Kafka топики
kafka-topics:
	@echo "Listing Kafka topics..."
	docker exec -it kafka kafka-topics --bootstrap-server localhost:9092 --list

## mongo-shell: Подключиться к MongoDB shell
mongo-shell:
	@echo "Connecting to MongoDB shell..."
	docker exec -it mongodb mongosh $(MONGO_URI) --authenticationDatabase admin

## mongo-query: Показать последние 10 уведомлений
mongo-query:
	@echo "Querying last 10 notifications..."
	docker exec -it mongodb mongosh $(MONGO_URI) --authenticationDatabase admin --eval 'use notification_db; db.large_transfers.find().sort({processed_at: -1}).limit(10).pretty()'

## mongo-count: Показать количество уведомлений
mongo-count:
	@echo "Counting notifications..."
	docker exec -it mongodb mongosh $(MONGO_URI) --authenticationDatabase admin --eval 'use notification_db; db.large_transfers.countDocuments()'

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

## logs: Показать логи приложения
logs:
	@tail -f notification.log

.DEFAULT_GOAL := help