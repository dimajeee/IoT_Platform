package httptransport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dmitrijsterligov/iot-platform/internal/domain"
)

type TelemetryQueryService interface {
	List(ctx context.Context, filter domain.TelemetryFilter) ([]domain.Telemetry, error)
	GetLatest(ctx context.Context, sensorID string) (domain.Telemetry, error)
	ListLatest(ctx context.Context) ([]domain.Telemetry, error)
}

type DeviceCommandService interface {
	SetInterval(ctx context.Context, interval string) error
}

type Server struct {
	addr           string
	logger         *slog.Logger
	service        TelemetryQueryService
	commandService DeviceCommandService
}

func NewServer(addr string, logger *slog.Logger, service TelemetryQueryService, commandService DeviceCommandService) *Server {
	return &Server{
		addr:           addr,
		logger:         logger,
		service:        service,
		commandService: commandService,
	}
}

func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.handleRoot)
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /api/v1/telemetry", s.handleListTelemetry)
	mux.HandleFunc("GET /api/v1/sensors/latest", s.handleListLatest)
	mux.HandleFunc("GET /api/v1/sensors/{sensorID}/latest", s.handleGetLatest)
	mux.HandleFunc("POST /api/v1/device/interval", s.handleSetInterval)
	mux.HandleFunc("GET /openapi.json", s.handleOpenAPI)
	mux.HandleFunc("GET /swagger", s.handleSwaggerUI)

	server := &http.Server{
		Addr:              s.addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)

	go func() {
		s.logger.Info("http server started", slog.String("addr", s.addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("listen and serve http: %w", err)
			return
		}

		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown http server: %w", err)
		}

		return nil
	case err := <-errCh:
		return err
	}
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/swagger", http.StatusTemporaryRedirect)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleListTelemetry(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid limit: %v", err))
			return
		}

		limit = parsed
	}

	items, err := s.service.List(r.Context(), domain.TelemetryFilter{
		SensorID:   strings.TrimSpace(r.URL.Query().Get("sensor_id")),
		SensorType: strings.TrimSpace(r.URL.Query().Get("sensor_type")),
		Limit:      limit,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"count": len(items),
	})
}

func (s *Server) handleListLatest(w http.ResponseWriter, r *http.Request) {
	items, err := s.service.ListLatest(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"count": len(items),
	})
}

func (s *Server) handleGetLatest(w http.ResponseWriter, r *http.Request) {
	sensorID := strings.TrimSpace(r.PathValue("sensorID"))
	item, err := s.service.GetLatest(r.Context(), sensorID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}

		if strings.Contains(err.Error(), "sensor id is empty") {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleSetInterval(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Interval string `json:"interval"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("decode request body: %v", err))
		return
	}

	if err := s.commandService.SetInterval(r.Context(), request.Interval); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":   "published",
		"interval": strings.TrimSpace(request.Interval),
	})
}

func (s *Server) handleOpenAPI(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, openAPISpec())
}

func (s *Server) handleSwaggerUI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(swaggerHTML))
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func openAPISpec() map[string]any {
	return map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       "IoT Telemetry Backend API",
			"version":     "1.0.0",
			"description": "REST API for reading telemetry history and latest sensor state.",
		},
		"servers": []map[string]any{
			{"url": "/"},
		},
		"paths": map[string]any{
			"/healthz": map[string]any{
				"get": map[string]any{
					"summary": "Health check",
					"responses": map[string]any{
						"200": map[string]any{
							"description": "Service is healthy",
						},
					},
				},
			},
			"/api/v1/telemetry": map[string]any{
				"get": map[string]any{
					"summary": "List telemetry history",
					"parameters": []map[string]any{
						queryParam("sensor_id", "string", "Filter by sensor id"),
						queryParam("sensor_type", "string", "Filter by sensor type"),
						queryParam("limit", "integer", "Maximum number of items to return"),
					},
					"responses": map[string]any{
						"200": map[string]any{
							"description": "Telemetry list",
						},
					},
				},
			},
			"/api/v1/sensors/latest": map[string]any{
				"get": map[string]any{
					"summary": "List latest telemetry for all sensors",
					"responses": map[string]any{
						"200": map[string]any{
							"description": "Latest sensor states",
						},
					},
				},
			},
			"/api/v1/sensors/{sensorID}/latest": map[string]any{
				"get": map[string]any{
					"summary": "Get latest telemetry by sensor id",
					"parameters": []map[string]any{
						{
							"name":     "sensorID",
							"in":       "path",
							"required": true,
							"schema": map[string]any{
								"type": "string",
							},
							"description": "Unique sensor identifier",
						},
					},
					"responses": map[string]any{
						"200": map[string]any{
							"description": "Latest sensor telemetry",
						},
						"404": map[string]any{
							"description": "Sensor telemetry not found",
						},
					},
				},
			},
			"/api/v1/device/interval": map[string]any{
				"post": map[string]any{
					"summary": "Set ESP32 telemetry publish interval",
					"requestBody": map[string]any{
						"required": true,
						"content": map[string]any{
							"application/json": map[string]any{
								"schema": map[string]any{
									"$ref": "#/components/schemas/SetIntervalRequest",
								},
								"examples": map[string]any{
									"seconds": map[string]any{
										"value": map[string]any{"interval": "2s"},
									},
									"minutes": map[string]any{
										"value": map[string]any{"interval": "3m"},
									},
									"hours": map[string]any{
										"value": map[string]any{"interval": "4h"},
									},
								},
							},
						},
					},
					"responses": map[string]any{
						"200": map[string]any{
							"description": "Interval command published",
						},
						"400": map[string]any{
							"description": "Invalid interval format",
						},
					},
				},
			},
		},
		"components": map[string]any{
			"schemas": map[string]any{
				"Telemetry": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"sensor_id":   map[string]any{"type": "string"},
						"sensor_type": map[string]any{"type": "string"},
						"value":       map[string]any{"type": "number"},
						"unit":        map[string]any{"type": "string"},
						"recorded_at": map[string]any{"type": "string", "format": "date-time"},
					},
				},
				"SetIntervalRequest": map[string]any{
					"type":     "object",
					"required": []string{"interval"},
					"properties": map[string]any{
						"interval": map[string]any{
							"type":        "string",
							"pattern":     "^[1-9][0-9]*(s|m|h)$",
							"example":     "2s",
							"description": "Interval with unit: s, m or h",
						},
					},
				},
			},
		},
	}
}

func queryParam(name, schemaType, description string) map[string]any {
	return map[string]any{
		"name":        name,
		"in":          "query",
		"required":    false,
		"description": description,
		"schema": map[string]any{
			"type": schemaType,
		},
	}
}

const swaggerHTML = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>IoT Telemetry API</title>
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
    <style>
      body { margin: 0; background: #f5f7fb; }
      .topbar { display: none; }
    </style>
  </head>
  <body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script>
      window.ui = SwaggerUIBundle({
        url: '/openapi.json',
        dom_id: '#swagger-ui',
        deepLinking: true,
        displayRequestDuration: true,
      });
    </script>
  </body>
</html>`
