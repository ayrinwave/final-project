# GW-Currency-Wallet

Микросервис для управления кошельком и обмена валют с поддержкой JWT авторизации.

## Функциональность

- 🔐 Регистрация и авторизация пользователей (JWT)
- 💰 Управление балансом в трёх валютах (USD, RUB, EUR)
- 💸 Пополнение и вывод средств
- 🔄 Обмен валют с актуальными курсами
- 📊 Идемпотентность операций (через request_id)
- 🔔 Отправка уведомлений о крупных переводах (≥30k) в Kafka

## Технологический стек

- **Go 1.21+**
- **PostgreSQL 16** - основная БД
- **gRPC** - коммуникация с exchanger сервисом
- **Kafka** - отправка событий о крупных переводах
- **JWT** - авторизация
- **Swagger** - API документация

## Архитектура
```
internal/
├── api/
│   ├── handlers/      # HTTP handlers
│   └── middlew/       # Middleware (auth, logging)
├── service/           # Бизнес-логика
├── storage/           # Data access layer
│   └── postgres/      # PostgreSQL реализация
├── grpc_client/       # gRPC клиент для exchanger
├── kafka/             # Kafka producer
└── models/            # Модели данных
```

## Быстрый старт

### Предварительные требования
```bash
# Установить зависимости
go install github.com/swaggo/swag/cmd/swag@latest
go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Или через Makefile
make deps
```

### 1. Запустить базы данных
```bash
# В корне проекта
docker-compose up -d postgres-wallet kafka zookeeper
```

### 2. Применить миграции
```bash
make migrate-up
```

### 3. Настроить конфигурацию

Создать `config.env`:
```env
# Application
APP_PORT=8080

# Database
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=postgres
POSTGRES_PASSWORD=1234
POSTGRES_DB=wallet
POSTGRES_SSLMODE=disable

# JWT
JWT_SECRET=your-super-secret-jwt-key-change-me-in-production
JWT_EXPIRATION=24h

# gRPC Exchanger Service
EXCHANGER_GRPC_ADDR=127.0.0.1:50051
GRPC_TIMEOUT=10s

# Kafka (для уведомлений о крупных переводах >= 30000)
KAFKA_ENABLED=true
KAFKA_BROKERS=localhost:9092
KAFKA_TOPIC=large-transfers
```

### 4. Запустить сервис
```bash
make run
```

Сервис будет доступен на `http://localhost:8080`

## API Endpoints

### Authentication

#### POST /api/v1/register
Регистрация нового пользователя

**Request:**
```json
{
  "username": "john_doe",
  "password": "securepass123",
  "email": "john@example.com"
}
```

**Response:** `201 Created`
```json
{
  "message": "User registered successfully"
}
```

#### POST /api/v1/login
Авторизация пользователя

**Request:**
```json
{
  "username": "john_doe",
  "password": "securepass123"
}
```

**Response:** `200 OK`
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

### Wallet Operations

> **Требуется авторизация:** `Authorization: Bearer <token>`

#### GET /api/v1/balance
Получить баланс пользователя

**Response:** `200 OK`
```json
{
  "balance": {
    "USD": 1000.50,
    "RUB": 50000.00,
    "EUR": 850.25
  }
}
```

#### POST /api/v1/wallet/deposit
Пополнить счёт

**Request:**
```json
{
  "amount": 100.00,
  "currency": "USD",
  "request_id": "unique-request-id-123"
}
```

**Response:** `200 OK`
```json
{
  "message": "Account topped up successfully",
  "new_balance": {
    "USD": 1100.50,
    "RUB": 50000.00,
    "EUR": 850.25
  }
}
```

#### POST /api/v1/wallet/withdraw
Вывести средства

**Request:**
```json
{
  "amount": 50.00,
  "currency": "USD",
  "request_id": "unique-request-id-456"
}
```

**Response:** `200 OK`
```json
{
  "message": "Withdrawal successful",
  "new_balance": {
    "USD": 1050.50,
    "RUB": 50000.00,
    "EUR": 850.25
  }
}
```

### Exchange Operations

#### GET /api/v1/exchange/rates
Получить актуальные курсы валют (публичный endpoint)

**Response:** `200 OK`
```json
{
  "rates": {
    "USD": 1.0,
    "RUB": 95.5,
    "EUR": 0.92
  }
}
```

#### POST /api/v1/exchange
Обменять валюту

**Request:**
```json
{
  "from_currency": "USD",
  "to_currency": "EUR",
  "amount": 100.00,
  "request_id": "unique-request-id-789"
}
```

**Response:** `200 OK`
```json
{
  "message": "Exchange successful",
  "exchanged_amount": 92.00,
  "rate": 0.92
}
```

## Swagger документация

После запуска сервиса документация доступна по адресу:
```
http://localhost:8080/swagger/index.html
```

Для обновления документации:
```bash
make swagger
```

## Разработка

### Запуск тестов
```bash
# Все тесты
make test

# Только unit тесты (без integration)
make test-short

# С покрытием
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Hot reload
```bash
# Требует установленный air: go install github.com/cosmtrek/air@latest
make dev
```

### Линтинг
```bash
make lint
make fmt
```

## Makefile команды
```bash
make help              # Показать все команды
make build             # Собрать бинарник
make run               # Запустить локально
make test              # Запустить тесты
make migrate-up        # Применить миграции
make migrate-down      # Откатить миграции
make migrate-create NAME=add_field  # Создать новую миграцию
make swagger           # Обновить Swagger документацию
make clean             # Очистить артефакты
```

## Структура БД

### Таблица `users`
- `id` UUID (PK)
- `username` VARCHAR(50) UNIQUE
- `email` VARCHAR(255) UNIQUE
- `password_hash` VARCHAR(255)
- `created_at` TIMESTAMPTZ
- `updated_at` TIMESTAMPTZ

### Таблица `wallets`
- `id` UUID (PK)
- `user_id` UUID (FK → users)
- `currency` VARCHAR(3)
- `balance` BIGINT (в минимальных единицах)
- `version` BIGINT (для optimistic locking)
- `created_at` TIMESTAMPTZ
- `updated_at` TIMESTAMPTZ
- UNIQUE(user_id, currency)

### Таблица `exchange_operations`
- `id` UUID (PK)
- `user_id` UUID (FK → users)
- `from_currency` VARCHAR(3)
- `to_currency` VARCHAR(3)
- `amount` BIGINT
- `exchanged_amount` BIGINT
- `rate` NUMERIC(20,10)
- `request_id` TEXT UNIQUE
- `created_at` TIMESTAMPTZ

## Идемпотентность

Все операции изменения баланса (deposit, withdraw, exchange) требуют уникальный `request_id`. Повторный запрос с тем же `request_id` вернёт `409 Conflict`.

**Пример:**
```bash
curl -X POST http://localhost:8080/api/v1/wallet/deposit \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "amount": 100,
    "currency": "USD",
    "request_id": "client-generated-uuid-123"
  }'
```

## Мониторинг и логи

Логи записываются в `wallet.log` и stdout в структурированном формате (JSON).
```bash
# Просмотр логов
make logs

# Или
tail -f wallet.log
```

## Troubleshooting

### Ошибка подключения к БД
```bash
# Проверить статус PostgreSQL
docker ps | grep postgres-wallet

# Проверить логи
docker logs postgres-wallet

# Пересоздать БД
docker-compose down -v
docker-compose up -d postgres-wallet
make migrate-up
```

### Ошибка подключения к exchanger

Убедись что `gw-exchanger` запущен на порту 50051:
```bash
# Проверить через grpcurl
grpcurl -plaintext localhost:50051 list
```

### Kafka недоступна
```bash
# Проверить статус
docker ps | grep kafka

# Проверить топики
docker exec -it kafka kafka-topics --bootstrap-server localhost:9092 --list
```

## Production considerations

- [ ] Изменить `JWT_SECRET` на сильный случайный ключ
- [ ] Настроить rate limiting для `/login` endpoint
- [ ] Добавить мониторинг (Prometheus + Grafana)
- [ ] Настроить log rotation для `wallet.log`
- [ ] Использовать connection pooling для gRPC
- [ ] Настроить graceful shutdown timeout
- [ ] Добавить health check endpoint
- [ ] Включить TLS для gRPC соединений

## License

MIT