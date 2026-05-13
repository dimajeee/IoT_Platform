package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dmitrijsterligov/iot-platform/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TelemetryRepository struct {
	pool *pgxpool.Pool
}

func NewTelemetryRepository(pool *pgxpool.Pool) *TelemetryRepository {
	return &TelemetryRepository{pool: pool}
}

func (r *TelemetryRepository) Save(ctx context.Context, telemetry domain.Telemetry) error {
	payload, err := json.Marshal(telemetry)
	if err != nil {
		return fmt.Errorf("marshal telemetry payload: %w", err)
	}

	const query = `
		INSERT INTO sensor_telemetry (sensor_id, sensor_type, value, unit, recorded_at, payload)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	if _, err := r.pool.Exec(
		ctx,
		query,
		telemetry.SensorID,
		telemetry.SensorType,
		telemetry.Value,
		telemetry.Unit,
		telemetry.RecordedAt,
		payload,
	); err != nil {
		return fmt.Errorf("insert telemetry: %w", err)
	}

	return nil
}

func (r *TelemetryRepository) List(ctx context.Context, filter domain.TelemetryFilter) ([]domain.Telemetry, error) {
	var (
		args       []any
		conditions []string
	)

	query := strings.Builder{}
	query.WriteString(`
		SELECT sensor_id, sensor_type, value, unit, recorded_at
		FROM sensor_telemetry
	`)

	if filter.SensorID != "" {
		args = append(args, filter.SensorID)
		conditions = append(conditions, fmt.Sprintf("sensor_id = $%d", len(args)))
	}

	if filter.SensorType != "" {
		args = append(args, filter.SensorType)
		conditions = append(conditions, fmt.Sprintf("sensor_type = $%d", len(args)))
	}

	if len(conditions) > 0 {
		query.WriteString(" WHERE ")
		query.WriteString(strings.Join(conditions, " AND "))
	}

	args = append(args, filter.Limit)
	query.WriteString(fmt.Sprintf(" ORDER BY recorded_at DESC LIMIT $%d", len(args)))

	rows, err := r.pool.Query(ctx, query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("query telemetry list: %w", err)
	}
	defer rows.Close()

	items := make([]domain.Telemetry, 0, filter.Limit)
	for rows.Next() {
		var item domain.Telemetry
		if err := rows.Scan(
			&item.SensorID,
			&item.SensorType,
			&item.Value,
			&item.Unit,
			&item.RecordedAt,
		); err != nil {
			return nil, fmt.Errorf("scan telemetry row: %w", err)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate telemetry rows: %w", err)
	}

	return items, nil
}
