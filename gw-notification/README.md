# GW-Notification

Микросервис для обработки и сохранения уведомлений о крупных денежных переводах (≥30,000).

## Функциональность

- 📨 Чтение сообщений из Kafka
- 💾 Сохранение уведомлений в MongoDB
- ⚡ Обработка до 1000 сообщений/сек
- 🔄 Идемпотентность (дубликаты игнорируются)
- 👷 Параллельная обработка через worker pool

## Технологический стек

- **Go 1.21+**
- **Kafka** - очередь сообщений
- **MongoDB 7** - хранение уведомлений
- **Sarama** - Kafka клиент

## Архитектура
```
internal/
├── kafka/             # Kafka consumer
├── storage/           # Data access layer
│   └── mongodb/       # MongoDB реализация
└── models/            # Модели данных
```

## Быстрый старт

### 1. Запустить зависимости
```bash
# В корне проекта
docker-compose up -d mongodb kafka zookeeper
```

Дождаться готовности:
```bash
docker ps  # Проверить что контейнеры UP
```

### 2. Настроить конфигурацию

Создать `config.env`:
```env
# Kafka Configuration
KAFKA_BROKERS=localhost:9092
KAFKA_TOPIC=large-transfers
KAFKA_GROUP_ID=notification-service
KAFKA_WORKERS=5
KAFKA_TIMEOUT=10s

# MongoDB Configuration
MONGO_URI=mongodb://admin:admin123@localhost:27017
MONGO_DATABASE=notifications
MONGO_COLLECTION=large_transfers
MONGO_TIMEOUT=10s
```

### 3. Запустить сервис
```bash
make run
```

Сервис начнёт читать сообщения из Kafka топика `large-transfers`.

## Формат сообщений

### Kafka Message (JSON)
```json
{
  "transaction_id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": "a1b2c3d4-e5f6-4789-a012-3456789abcde",
  "from_currency": "USD",
  "to_currency": "EUR",
  "amount": 35000.00,
  "exchanged_amount": 32200.00,
  "rate": 0.92,
  "timestamp": "2025-01-15T14:30:00Z"
}
```

### MongoDB Document
```json
{
  "_id": ObjectId("..."),
  "transaction_id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": "a1b2c3d4-e5f6-4789-a012-3456789abcde",
  "from_currency": "USD",
  "to_currency": "EUR",
  "amount": 35000.00,
  "exchanged_amt": 32200.00,
  "rate": 0.92,
  "timestamp": ISODate("2025-01-15T14:30:00Z"),
  "processed_at": ISODate("2025-01-15T14:30:05Z")
}
```

**Индексы:**
- Unique index на `transaction_id` (для идемпотентности)

## Использование

### Отправить тестовое сообщение
```bash
make kafka-test
```

Это отправит тестовое сообщение о крупном переводе в Kafka.

### Проверить сохранённые уведомления
```bash
# Последние 10 уведомлений
make mongo-query

# Количество уведомлений
make mongo-count

# Подключиться к MongoDB shell
make mongo-shell
```

### MongoDB queries
```javascript
// Подключиться
use notification_db

// Все уведомления
db.large_transfers.find().pretty()

// Последние 10
db.large_transfers.find().sort({processed_at: -1}).limit(10)

// По user_id
db.large_transfers.find({user_id: "a1b2c3d4-..."})

// По диапазону дат
db.large_transfers.find({
  timestamp: {
    $gte: ISODate("2025-01-01"),
    $lt: ISODate("2025-02-01")
  }
})

// Сумма всех переводов
db.large_transfers.aggregate([
  {$group: {_id: null, total: {$sum: "$amount"}}}
])
```

## Разработка

### Worker pool

Сервис использует worker pool для параллельной обработки сообщений:
```go
KAFKA_WORKERS=5  // Количество воркеров
```

Каждый воркер:
1. Читает сообщение из Kafka
2. Десериализует JSON
3. Сохраняет в MongoDB
4. Помечает сообщение как обработанное (commit offset)

### Идемпотентность

Duplicate messages автоматически игнорируются благодаря unique index на `transaction_id`:
```go
if mongo.IsDuplicateKeyError(err) {
    return nil  // Уже сохранено ранее
}
```

### Graceful shutdown

При получении SIGINT/SIGTERM:
1. Останавливается чтение новых сообщений
2. Обрабатываются текущие сообщения
3. Закрывается Kafka consumer
4. Закрывается MongoDB connection

## Тестирование

### Unit тесты
```bash
make test
```

### Integration тест
```bash
# 1. Запустить все зависимости
docker-compose up -d

# 2. Запустить сервис
make run

# 3. В другом терминале отправить тестовое сообщение
make kafka-test

# 4. Проверить что сообщение сохранилось
make mongo-query
```

### Kafka debugging
```bash
# Читать сообщения из топика
make kafka-consume

# Список всех топиков
make kafka-topics

# Логи Kafka
docker logs -f kafka
```

## Makefile команды
```bash
make help              # Показать все команды
make build             # Собрать бинарник
make run               # Запустить локально
make test              # Запустить тесты
make kafka-test        # Отправить тестовое сообщение
make kafka-consume     # Читать сообщения из Kafka
make kafka-topics      # Список Kafka топиков
make mongo-shell       # MongoDB shell
make mongo-query       # Последние 10 уведомлений
make mongo-count       # Количество уведомлений
make logs              # Показать логи сервиса
make clean             # Очистить артефакты
```

## Мониторинг и логи

### Логи приложения

Логи записываются в `notification.log` и stdout в структурированном формате.
```bash
# Просмотр логов
make logs

# Или
tail -f notification.log
```

**Уровни логов:**
- `DEBUG` - получение сообщений из Kafka
- `INFO` - успешное сохранение уведомления
- `ERROR` - ошибки десериализации или сохранения

### Метрики

Логи содержат информацию для мониторинга:
- Transaction ID
- User ID
- Amount
- Currencies
- Processing time

**Пример лога:**
```json
{
  "time": "2025-01-15T14:30:05Z",
  "level": "INFO",
  "msg": "уведомление успешно сохранено",
  "transaction_id": "550e8400-...",
  "user_id": "a1b2c3d4-...",
  "amount": 35000.0,
  "from": "USD",
  "to": "EUR"
}
```

## Troubleshooting

### Consumer не получает сообщения
```bash
# 1. Проверить что Kafka работает
docker ps | grep kafka

# 2. Проверить что топик существует
make kafka-topics

# 3. Проверить логи Kafka
docker logs kafka

# 4. Отправить тестовое сообщение
make kafka-test

# 5. Читать сообщения вручную
make kafka-consume
```

### Ошибка подключения к MongoDB
```bash
# Проверить MongoDB
docker ps | grep mongodb
docker logs mongodb

# Проверить credentials
docker exec -it mongodb mongosh mongodb://admin:admin123@localhost:27017 --authenticationDatabase admin
```

### Consumer lag (отставание обработки)

Если сообщения накапливаются быстрее чем обрабатываются:
```env
# Увеличить количество воркеров
KAFKA_WORKERS=10
```

Или масштабировать horizontally:
```bash
# Запустить несколько инстансов с одним group_id
./notification-service &
./notification-service &
./notification-service &
```

Kafka автоматически распределит партиции между инстансами.

## Production considerations

- [ ] Настроить Kafka retention policy для топика
- [ ] Добавить Dead Letter Queue для poison messages
- [ ] Настроить MongoDB replica set для HA
- [ ] Добавить Prometheus метрики (messages/sec, errors/sec, lag)
- [ ] Настроить log rotation для `notification.log`
- [ ] Добавить alerting при ошибках сохранения
- [ ] Настроить Kafka consumer max.poll.interval
- [ ] Добавить circuit breaker для MongoDB
- [ ] Создать backup strategy для MongoDB
- [ ] Мониторить consumer lag через Kafka Manager

## License

MIT