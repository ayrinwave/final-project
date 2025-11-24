.PHONY: help build-all clean-all proto db-up db-down db-status logs-wallet logs-exchanger logs-notification test-all

## help: Показать help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## db-up: Запустить все БД через docker-compose
db-up:
	@echo "Starting databases..."
	docker-compose up -d
	@echo "Databases started. Waiting for health checks..."
	@sleep 5
	@echo "Databases ready!"

## db-down: Остановить все БД
db-down:
	@echo "Stopping databases..."
	docker-compose down
	@echo "Databases stopped"

## db-clean: Остановить БД и удалить volumes
db-clean:
	@echo "Stopping databases and removing volumes..."
	docker-compose down -v
	@echo "Databases cleaned"

## db-status: Показать статус БД
db-status:
	@echo "Database status:"
	@docker-compose ps

## db-logs: Показать логи БД
db-logs:
	@docker-compose logs -f

## build-all: Собрать все сервисы
build-all:
	@echo "Building all services..."
	@cd gw-exchanger && make build
	@cd gw-currency-wallet && make build
	@cd gw-notification && make build
	@echo "All services built"

## run-exchanger: Запустить gw-exchanger
run-exchanger:
	@echo "Starting gw-exchanger..."
	@cd gw-exchanger && make run

## run-wallet: Запустить gw-currency-wallet
run-wallet:
	@echo "Starting gw-currency-wallet..."
	@cd gw-currency-wallet && make run

## run-notification: Запустить gw-notification
run-notification:
	@echo "Starting gw-notification..."
	@cd gw-notification && make run

## migrate-all: Применить миграции для всех сервисов
migrate-all:
	@echo "Running migrations for all services..."
	@cd gw-exchanger && make migrate-up
	@cd gw-currency-wallet && make migrate-up
	@echo "All migrations applied"

## clean-all: Очистить все сервисы
clean-all:
	@echo "Cleaning all services..."
	@cd gw-exchanger && make clean
	@cd gw-currency-wallet && make clean
	@cd gw-notification && make clean
	@echo "All services cleaned"

## proto: Сгенерировать protobuf для exchanger
proto:
	@echo "Generating protobuf..."
	@cd gw-exchanger && make proto
	@echo "Protobuf generated"

## test-all: Запустить тесты всех сервисов
test-all:
	@echo "Running tests for all services..."
	@cd gw-exchanger && make test
	@cd gw-currency-wallet && make test
	@cd gw-notification && make test
	@echo "All tests completed"

## logs-wallet: Показать логи wallet
logs-wallet:
	@tail -f gw-currency-wallet/wallet.log

## logs-exchanger: Показать логи exchanger
logs-exchanger:
	@tail -f gw-exchanger/exchanger.log

## logs-notification: Показать логи notification
logs-notification:
	@tail -f gw-notification/notification.log

## setup: Первоначальная настройка (БД + миграции)
setup: db-up
	@echo "Waiting for databases to be ready..."
	@sleep 10
	@make migrate-all
	@echo "Setup complete! You can now run services with 'make run-<service>'"

.DEFAULT_GOAL := help