CREATE TABLE IF NOT EXISTS sensor_telemetry (
    id BIGSERIAL PRIMARY KEY,
    sensor_id TEXT NOT NULL,
    sensor_type TEXT NOT NULL,
    value DOUBLE PRECISION NOT NULL,
    unit TEXT NOT NULL,
    recorded_at TIMESTAMPTZ NOT NULL,
    payload JSONB NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_sensor_telemetry_sensor_recorded_at
    ON sensor_telemetry (sensor_id, recorded_at DESC);
