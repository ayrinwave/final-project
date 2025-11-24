# GW-Exchanger

gRPC микросервис для предоставления актуальных курсов валют.

## Функциональность

- 📊 Получение курсов всех валют (USD, RUB, EUR)
- 🔄 Получение курса обмена между двумя валютами
- 💾 Хранение курсов в PostgreSQL
- ⚡ Быстрая отдача данных через gRPC

## Технологический стек

- **Go 1.21+**
- **PostgreSQL 16** - хранение курсов валют
- **gRPC** - API протокол
- **Protocol Buffers** - сериализация данных

## Архитектура
```
internal/
├── grpc_server/       # gRPC server реализация
├── storage/           # Data access layer
│   └── postgres/      # PostgreSQL реализация
└── models/            # Модели данных
```

## Быстрый старт

### 1. Запустить PostgreSQL
```bash
# В корне проекта
docker-compose up -d postgres-exchanger
```

### 2. Применить миграции
```bash
make migrate-up
```

Миграция создаст таблицу `exchange_rates` и заполнит начальными данными:
- USD: 1.0 (базовая валюта)
- RUB: 95.5
- EUR: 0.92

### 3. Настроить конфигурацию

Создать `config.env`:
```env
# gRPC Server
GRPC_PORT=50051

# Database
POSTGRES_HOST=localhost
POSTGRES_PORT=5433
POSTGRES_USER=postgres
POSTGRES_PASSWORD=1234
POSTGRES_DB=exchanger
POSTGRES_SSLMODE=disable
```

### 4. Запустить сервис
```bash
make run
```

gRPC сервер будет доступен на `localhost:50051`

## gRPC API

### Service: ExchangeService

#### GetExchangeRates()

Получить курсы всех валют относительно USD.

**Request:** `Empty`

**Response:**
```protobuf
message ExchangeRatesResponse {
  map<string, double> rates = 1;
}
```

**Пример (grpcurl):**
```bash
grpcurl -plaintext localhost:50051 exchange.ExchangeService/GetExchangeRates
```

**Response:**
```json
{
  "rates": {
    "USD": 1.0,
    "RUB": 95.5,
    "EUR": 0.92
  }
}
```

#### GetExchangeRateForCurrency(from, to)

Получить курс обмена между двумя валютами.

**Request:**
```protobuf
message CurrencyRequest {
  string from_currency = 1;
  string to_currency = 2;
}
```

**Response:**
```protobuf
message ExchangeRateResponse {
  string from_currency = 1;
  string to_currency = 2;
  double rate = 3;
}
```

**Пример (grpcurl):**
```bash
grpcurl -plaintext \
  -d '{"from_currency":"USD","to_currency":"EUR"}' \
  localhost:50051 \
  exchange.ExchangeService/GetExchangeRateForCurrency
```

**Response:**
```json
{
  "fromCurrency": "USD",
  "toCurrency": "EUR",
  "rate": 0.92
}
```

## Protobuf схема
```protobuf
syntax = "proto3";

package exchange;

option go_package = "github.com/gw-exchanger/proto-exchange";

service ExchangeService {
    rpc GetExchangeRates(Empty) returns (ExchangeRatesResponse);
    rpc GetExchangeRateForCurrency(CurrencyRequest) returns (ExchangeRateResponse);
}

message Empty {}

message CurrencyRequest {
    string from_currency = 1;
    string to_currency = 2;
}

message ExchangeRateResponse {
    string from_currency = 1;
    string to_currency = 2;
    double rate = 3;
}

message ExchangeRatesResponse {
    map<string, double> rates = 1;
}
```

## Управление курсами

### Обновление курсов в БД
```sql
-- Подключиться к БД
psql -h localhost -p 5433 -U postgres -d exchanger

-- Обновить курс
UPDATE exchange_rates 
SET rate = 96.0, updated_at = NOW() 
WHERE currency = 'RUB';

-- Проверить текущие курсы
SELECT * FROM exchange_rates ORDER BY currency;
```

### Добавление новой валюты
```sql
INSERT INTO exchange_rates (currency, rate) 
VALUES ('GBP', 0.79);
```

## Разработка

### Генерация Protobuf кода

После изменения `.proto` файла:
```bash
make proto
```

Это сгенерирует:
- `exchange.pb.go` - типы сообщений
- `exchange_grpc.pb.go` - gRPC сервер/клиент интерфейсы

### Тестирование через grpcurl
```bash
# Установить grpcurl
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# Список сервисов
grpcurl -plaintext localhost:50051 list

# Список методов
grpcurl -plaintext localhost:50051 list exchange.ExchangeService

# Вызов метода
grpcurl -plaintext localhost:50051 exchange.ExchangeService/GetExchangeRates
```

### Тесты
```bash
make test
```

## Makefile команды
```bash
make help              # Показать все команды
make build             # Собрать бинарник
make run               # Запустить локально
make test              # Запустить тесты
make proto             # Сгенерировать protobuf код
make proto-clean       # Удалить сгенерированные файлы
make migrate-up        # Применить миграции
make migrate-down      # Откатить миграции
make grpc-test         # Протестировать через grpcurl
make clean             # Очистить артефакты
```

## Структура БД

### Таблица `exchange_rates`
- `id` SERIAL (PK)
- `currency` VARCHAR(3) UNIQUE
- `rate` DOUBLE PRECISION (относительно USD)
- `updated_at` TIMESTAMPTZ

**Индексы:**
- `idx_exchange_rates_currency` на `currency`

**Триггеры:**
- Автоматическое обновление `updated_at` при UPDATE

## Логика расчёта курсов

Курсы хранятся относительно USD (базовая валюта = 1.0).

**Пример расчёта USD → EUR:**
```
rate(USD→EUR) = rate(EUR) / rate(USD) = 0.92 / 1.0 = 0.92
```

**Пример расчёта RUB → EUR:**
```
rate(RUB→EUR) = rate(EUR) / rate(RUB) = 0.92 / 95.5 = 0.00963
```

## Мониторинг и логи

Логи записываются в `exchanger.log` и stdout.
```bash
# Просмотр логов
tail -f exchanger.log
```

**Формат логов:**
- `INFO` - успешные операции
- `ERROR` - ошибки БД или gRPC

## Troubleshooting

### gRPC server не запускается
```bash
# Проверить занят ли порт 50051
lsof -i :50051

# Или изменить порт в config.env
GRPC_PORT=50052
```

### Ошибка подключения к БД
```bash
# Проверить PostgreSQL
docker ps | grep postgres-exchanger
docker logs postgres-exchanger

# Пересоздать БД
docker-compose down -v
docker-compose up -d postgres-exchanger
make migrate-up
```

### protoc не найден
```bash
# Установить protoc
# macOS
brew install protobuf

# Linux
sudo apt install -y protobuf-compiler

# Или скачать с https://github.com/protocolbuffers/protobuf/releases

# Установить Go плагины
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

## Production considerations

- [ ] Добавить кэширование курсов (TTL 1-5 минут)
- [ ] Настроить TLS для gRPC
- [ ] Добавить health check endpoint
- [ ] Настроить connection pool для БД (MaxConns=20)
- [ ] Добавить метрики (requests/sec, latency)
- [ ] Настроить graceful shutdown с timeout
- [ ] Реплицировать БД для read-only запросов
- [ ] Добавить rate limiting (если публичный API)

## License

MIT