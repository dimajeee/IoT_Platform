# Hardware Integration

## Source Hardware

The analyzed DOCX describes an ESP32-based indoor microclimate monitoring prototype.

Hardware composition:

- ESP32 microcontroller
- DS18B20 temperature sensor, GPIO 5, 1-Wire
- DHT22 humidity sensor, GPIO 4
- SCD30 CO2 sensor, I2C
- SCD30 SDA: GPIO 21
- SCD30 SCL: GPIO 22
- data output through UART/USB serial
- measurement interval: about 5 seconds

## Integration Decision

The backend already uses MQTT as its ingestion boundary. The hardware prototype currently outputs measurements to UART, so the clean integration path is:

```text
ESP32 UART -> hardware-gateway -> Mosquitto MQTT -> backend -> PostgreSQL + Redis -> REST API + Swagger UI
```

This avoids coupling the backend to USB/serial hardware and keeps the backend deployable on a server where the ESP32 is not physically attached.

## MQTT Contract

The current external broker publishes ESP32 values to:

```text
/esp32/<sensor>/<metric>
```

Examples:

```text
/esp32/DS18B20/temperature 25.25
/esp32/DHT22/humidity 46.50
/esp32/SCD30/ppm 722.9
```

Sensor mapping:

| Hardware sensor | Backend sensor_id | sensor_type | unit |
|---|---|---|---|
| DS18B20 | `temperature-sensor-1` | `temperature` | `C` |
| DHT22 | `humidity-sensor-1` | `humidity` | `%` |
| SCD30 | `co2-sensor-1` | `co2` | `ppm` |

Status topics such as `/esp32/SCD30/status` are skipped by backend and are not stored as telemetry.

## Recommended ESP32 Output

The most reliable firmware output is one JSON object per line:

```json
{"sensor_id":"temperature-sensor-1","sensor_type":"temperature","value":24.6,"unit":"C"}
{"sensor_id":"humidity-sensor-1","sensor_type":"humidity","value":41.2,"unit":"%"}
{"sensor_id":"co2-sensor-1","sensor_type":"co2","value":658,"unit":"ppm"}
```

Text output is also supported:

```text
DS18B20 Temperature: 24.6 C
DHT22 Humidity: 41.2 %
SCD30 CO2: 658 ppm
```

## Running With Hardware

Linux with the same external MQTT broker as backend:

```bash
MQTT_HOST=broker.example.com \
MQTT_PORT=1883 \
MQTT_USERNAME=iot_user \
MQTT_PASSWORD=change_me \
HARDWARE_SERIAL_PORT=/dev/ttyUSB0 \
HARDWARE_SERIAL_BAUD=115200 \
docker compose --profile hardware up --build
```

If the ESP32 appears as `/dev/ttyACM0`:

```bash
HARDWARE_SERIAL_PORT=/dev/ttyACM0 HARDWARE_SERIAL_BAUD=115200 docker compose --profile hardware up --build
```

After data starts flowing, inspect:

```text
http://localhost:8080/swagger
```

Useful API calls:

```text
GET /api/v1/sensors/latest
GET /api/v1/sensors/temperature-sensor-1/latest
GET /api/v1/sensors/humidity-sensor-1/latest
GET /api/v1/sensors/co2-sensor-1/latest
GET /api/v1/telemetry?limit=20
```
