# IoT Platform Skeleton

## Стек

- Go
- MQTT broker: Mosquitto
- PostgreSQL
- Redis
- Docker Compose

## Структура

```text
cmd/backend/main.go
cmd/gateway/main.go
internal/app
internal/cache/redis
internal/config
internal/device
internal/domain
internal/repository/postgres
internal/storage/postgres
internal/storage/redis
internal/transport/mqtt
internal/usecase
internal/usecase/telemetry
deploy/mosquitto
deploy/postgres/init
build/backend
build/gateway
```

## Требования

Для запуска нужны:

- Docker
- Docker Compose

## Как собрать и запустить

1. Перейдите в каталог проекта:

```bash
cd (ввести папку проекта)
```

2. Поднимите сервисы:

```bash
docker compose up --build
```

После запуска поднимутся:

- `mosquitto` на `localhost:1883`
- `postgres` на `localhost:5432`
- `redis` на `localhost:6379`
- `backend api` на `localhost:8080`
- `backend`
- `gateway`

## Что должно происходить

- `gateway` раз в 5 секунд публикует telemetry от:
  `temperature-sensor-1`, `humidity-sensor-1`, `co2-sensor-1`
- `backend` получает сообщения из MQTT
- `backend` пишет каждое сообщение в таблицу `sensor_telemetry`
- `backend` обновляет Redis key формата `sensor:<sensor_id>:latest_state`
- `backend` отдает данные через REST API

## Swagger UI

После запуска откройте:

```text
http://localhost:8080/swagger
```

OpenAPI JSON доступен по адресу:

```text
http://localhost:8080/openapi.json
```

## REST API

### Health

```text
GET /healthz
```

### История телеметрии

```text
GET /api/v1/telemetry
GET /api/v1/telemetry?limit=20
GET /api/v1/telemetry?sensor_id=temperature-sensor-1
GET /api/v1/telemetry?sensor_type=co2
```

### Последние состояния

```text
GET /api/v1/sensors/latest
GET /api/v1/sensors/temperature-sensor-1/latest
GET /api/v1/sensors/humidity-sensor-1/latest
GET /api/v1/sensors/co2-sensor-1/latest
```

## Как проверить работу

### Логи

Откройте логи всех сервисов:

```bash
docker compose logs -f
```

Отдельно backend:

```bash
docker compose logs -f backend
```

Отдельно gateway:

```bash
docker compose logs -f gateway
```

### Проверка PostgreSQL

Посмотреть сохранённые telemetry записи:

```bash
docker compose exec postgres psql -U iot -d iot -c "SELECT sensor_id, sensor_type, value, unit, recorded_at FROM sensor_telemetry ORDER BY recorded_at DESC LIMIT 10;"
```

### Проверка Redis

Посмотреть последнее состояние устройства:

```bash
docker compose exec redis redis-cli GET sensor:temperature-sensor-1:latest_state
```

### Проверка MQTT вручную

Подписаться на telemetry topic:

```bash
docker compose exec mosquitto mosquitto_sub -h localhost -t 'iot/sensors/+/telemetry' -v
```

## Остановка

Остановить сервисы:

```bash
docker compose down
```

Остановить и удалить volume PostgreSQL:

```bash
docker compose down -v
```

## Локальная сборка бинарников

Если Go установлен локально:

```bash
go build ./cmd/backend
go build ./cmd/gateway
```

## Переменные окружения

### Backend

- `BACKEND_HTTP_ADDR`
- `BACKEND_MQTT_BROKER`
- `BACKEND_MQTT_CLIENT_ID`
- `BACKEND_MQTT_TOPIC`
- `BACKEND_POSTGRES_DSN`
- `BACKEND_REDIS_ADDR`
- `BACKEND_REDIS_PASSWORD`
- `BACKEND_REDIS_DB`
- `BACKEND_LOG_LEVEL`

### Gateway

- `GATEWAY_MQTT_BROKER`
- `GATEWAY_MQTT_CLIENT_ID`
- `GATEWAY_PUBLISH_INTERVAL`
- `GATEWAY_LOG_LEVEL`

## Примечания

- MQTT handler в backend только декодирует сообщение и передаёт его в usecase.
- Бизнес-логика сохранения вынесена в `internal/usecase/telemetry`.
- HTTP handler в backend только обрабатывает REST-запрос и вызывает query service.
- Ошибки оборачиваются через `fmt.Errorf(...: %w, err)`.
- Для PostgreSQL используется `pgx/v5`.
- Для Redis используется `go-redis/v9`.
- Для MQTT используется `eclipse/paho.mqtt.golang`.
