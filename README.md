# IoT Platform Skeleton

## Стек

- Go
- External MQTT broker
- PostgreSQL
- Redis
- Docker Compose

## Структура

```text
cmd/backend/main.go
cmd/gateway/main.go
cmd/hardware-gateway/main.go
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
build/hardware-gateway
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

- `postgres` на `localhost:5432`
- `redis` на `localhost:6379`
- `backend api` на `localhost:8080`
- `backend`

MQTT broker теперь внешний. Его host, port, username и password задаются в `.env`.

## MQTT config

Создайте `.env` из примера:

```bash
cp .env.example .env
```

Заполните параметры внешнего брокера:

```dotenv
MQTT_HOST=broker.example.com
MQTT_PORT=1883
MQTT_USERNAME=iot_backend_user
MQTT_PASSWORD=change_me
MQTT_TOPIC=/esp32/+/+
MQTT_DECODER=esp32-topic-value
MQTT_SENSOR_ID_TOPIC_POS=-1
```

Сейчас backend настроен под фактическую схему ESP32:

```text
/esp32/DS18B20/status 1
/esp32/DS18B20/temperature 25.25
/esp32/SCD30/status 1
/esp32/SCD30/ppm 722.9
/esp32/DHT22/status 1
/esp32/DHT22/humidity 46.50
```

`MQTT_DECODER=esp32-topic-value` означает, что backend берет тип датчика из topic, значение из payload, а `status`-топики пропускает. В API данные сохраняются как:

```text
DS18B20 temperature -> temperature-sensor-1, temperature, C
DHT22 humidity -> humidity-sensor-1, humidity, %
SCD30 ppm -> co2-sensor-1, co2, ppm
```

Для старого JSON-формата можно переключить decoder:

```dotenv
MQTT_TOPIC=iot/sensors/+/telemetry
MQTT_DECODER=json
MQTT_SENSOR_ID_TOPIC_POS=2
```

В этом режиме `MQTT_SENSOR_ID_TOPIC_POS` указывает номер сегмента topic, где лежит `sensor_id`.

## Подключение реального hardware

По документу `КНИР3_БИСТ-22-ИТ-1_Медведев_ДР.docx` прототип состоит из:

- ESP32
- DS18B20 для температуры, GPIO 5, интерфейс 1-Wire
- DHT22 для влажности, GPIO 4
- SCD30 для CO2, I2C: SDA GPIO 21, SCL GPIO 22
- вывод данных через UART примерно раз в 5 секунд

Backend уже принимает данные через MQTT. Если ESP32 публикует в брокер напрямую по topic `/esp32/<sensor>/<metric>`, `hardware-gateway` не нужен. Если ESP32 только пишет данные в UART, тогда `hardware-gateway` читает UART и публикует сообщения в MQTT.

Обычный запуск с тестовым генератором:

```bash
docker compose --profile simulator up --build
```

Запуск с реальным ESP32 через USB/UART:

```bash
HARDWARE_SERIAL_PORT=/dev/ttyUSB0 HARDWARE_SERIAL_BAUD=115200 docker compose --profile hardware up --build
```

Если ESP32 определяется иначе, сначала посмотрите доступные устройства:

```bash
ls -la /dev/ttyUSB* /dev/ttyACM*
```

На macOS serial-порт обычно выглядит так:

```bash
ls -la /dev/tty.usbserial* /dev/tty.usbmodem*
```

`hardware-gateway` понимает два формата вывода.

Рекомендуемый JSON-формат:

```json
{"sensor_id":"temperature-sensor-1","sensor_type":"temperature","value":24.6,"unit":"C"}
{"sensor_id":"humidity-sensor-1","sensor_type":"humidity","value":41.2,"unit":"%"}
{"sensor_id":"co2-sensor-1","sensor_type":"co2","value":658,"unit":"ppm"}
```

Также поддерживаются обычные текстовые строки:

```text
DS18B20 Temperature: 24.6 C
DHT22 Humidity: 41.2 %
SCD30 CO2: 658 ppm
```

## Что должно происходить

- `gateway` раз в 5 секунд публикует telemetry от:
  `temperature-sensor-1`, `humidity-sensor-1`, `co2-sensor-1`
- `hardware-gateway` при включенном профиле `hardware` публикует реальные данные ESP32
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

### Управление частотой обновления ESP32

Команда публикуется в MQTT topic `/esp32/interval`.

```text
POST /api/v1/device/interval
Content-Type: application/json

{"interval":"2s"}
```

Поддерживаемые единицы: `s`, `m`, `h`. Примеры: `2s`, `3m`, `4h`.

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

Подписаться на telemetry topic можно на стороне внешнего брокера любым MQTT-клиентом:

```bash
mosquitto_sub -h "$MQTT_HOST" -p "$MQTT_PORT" -u "$MQTT_USERNAME" -P "$MQTT_PASSWORD" -t '/esp32/+/+' -v
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
go build ./cmd/hardware-gateway
```

## Переменные окружения

### Backend

- `BACKEND_HTTP_ADDR`
- `BACKEND_MQTT_HOST`
- `BACKEND_MQTT_PORT`
- `BACKEND_MQTT_USERNAME`
- `BACKEND_MQTT_PASSWORD`
- `BACKEND_MQTT_CLIENT_ID`
- `BACKEND_MQTT_TOPIC`
- `BACKEND_MQTT_DECODER`
- `BACKEND_MQTT_SENSOR_ID_TOPIC_POS`
- `BACKEND_MQTT_COMMAND_TOPIC`
- `BACKEND_POSTGRES_DSN`
- `BACKEND_REDIS_ADDR`
- `BACKEND_REDIS_PASSWORD`
- `BACKEND_REDIS_DB`
- `BACKEND_LOG_LEVEL`

### Gateway

- `GATEWAY_MQTT_HOST`
- `GATEWAY_MQTT_PORT`
- `GATEWAY_MQTT_USERNAME`
- `GATEWAY_MQTT_PASSWORD`
- `GATEWAY_MQTT_CLIENT_ID`
- `GATEWAY_PUBLISH_INTERVAL`
- `GATEWAY_LOG_LEVEL`

### Hardware Gateway

- `HARDWARE_GATEWAY_MQTT_HOST`
- `HARDWARE_GATEWAY_MQTT_PORT`
- `HARDWARE_GATEWAY_MQTT_USERNAME`
- `HARDWARE_GATEWAY_MQTT_PASSWORD`
- `HARDWARE_GATEWAY_MQTT_CLIENT_ID`
- `HARDWARE_GATEWAY_SERIAL_PORT`
- `HARDWARE_GATEWAY_SERIAL_BAUD`
- `HARDWARE_GATEWAY_LOG_LEVEL`

## Примечания

- MQTT handler в backend только декодирует сообщение и передаёт его в usecase.
- Бизнес-логика сохранения вынесена в `internal/usecase/telemetry`.
- HTTP handler в backend только обрабатывает REST-запрос и вызывает query service.
- Ошибки оборачиваются через `fmt.Errorf(...: %w, err)`.
- Для PostgreSQL используется `pgx/v5`.
- Для Redis используется `go-redis/v9`.
- Для MQTT используется `eclipse/paho.mqtt.golang`.
