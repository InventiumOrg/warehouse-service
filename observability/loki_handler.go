package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"
)

// LokiHandler implements slog.Handler to send logs directly to Loki
type LokiHandler struct {
	client   *http.Client
	lokiURL  string
	labels   map[string]string
	level    slog.Level
	fallback slog.Handler // Fallback to stdout if Loki is unavailable
}

// LokiConfig holds configuration for Loki handler
type LokiConfig struct {
	URL    string
	Labels map[string]string
	Level  slog.Level
}

// NewLokiHandler creates a new Loki handler
func NewLokiHandler(config LokiConfig) *LokiHandler {
	if config.Labels == nil {
		config.Labels = make(map[string]string)
	}

	// Add default labels
	if config.Labels["service"] == "" {
		config.Labels["service"] = "warehouse-service"
	}
	if config.Labels["job"] == "" {
		config.Labels["job"] = "go-app"
	}

	return &LokiHandler{
		client:   &http.Client{Timeout: 5 * time.Second},
		lokiURL:  config.URL + "/loki/api/v1/push",
		labels:   config.Labels,
		level:    config.Level,
		fallback: slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: config.Level}),
	}
}

// Enabled reports whether the handler handles records at the given level
func (h *LokiHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

// Handle processes a log record
func (h *LokiHandler) Handle(ctx context.Context, record slog.Record) error {
	// Also send to fallback (stdout)
	if err := h.fallback.Handle(ctx, record); err != nil {
		return err
	}

	// Send to Loki asynchronously to avoid blocking
	go h.sendToLoki(record)
	return nil
}

// WithAttrs returns a new handler with additional attributes
func (h *LokiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newLabels := make(map[string]string)
	for k, v := range h.labels {
		newLabels[k] = v
	}

	// Add attrs as labels (only string values)
	for _, attr := range attrs {
		if attr.Value.Kind() == slog.KindString {
			newLabels[attr.Key] = attr.Value.String()
		}
	}

	return &LokiHandler{
		client:   h.client,
		lokiURL:  h.lokiURL,
		labels:   newLabels,
		level:    h.level,
		fallback: h.fallback.WithAttrs(attrs),
	}
}

// WithGroup returns a new handler with a group
func (h *LokiHandler) WithGroup(name string) slog.Handler {
	return &LokiHandler{
		client:   h.client,
		lokiURL:  h.lokiURL,
		labels:   h.labels,
		level:    h.level,
		fallback: h.fallback.WithGroup(name),
	}
}

// LokiPayload represents the Loki push API payload
type LokiPayload struct {
	Streams []LokiStream `json:"streams"`
}

// LokiStream represents a log stream in Loki
type LokiStream struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}

// sendToLoki sends the log record to Loki
func (h *LokiHandler) sendToLoki(record slog.Record) {
	// Build the log entry
	logEntry := map[string]interface{}{
		"timestamp": record.Time.Format(time.RFC3339Nano),
		"level":     record.Level.String(),
		"msg":       record.Message,
	}

	// Add all attributes
	record.Attrs(func(attr slog.Attr) bool {
		logEntry[attr.Key] = attr.Value.Any()
		return true
	})

	// Convert to JSON
	logJSON, err := json.Marshal(logEntry)
	if err != nil {
		return // Silently fail to avoid log loops
	}

	// Create Loki payload
	payload := LokiPayload{
		Streams: []LokiStream{
			{
				Stream: h.labels,
				Values: [][]string{
					{
						strconv.FormatInt(record.Time.UnixNano(), 10),
						string(logJSON),
					},
				},
			},
		},
	}

	// Send to Loki
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return
	}

	req, err := http.NewRequest("POST", h.lokiURL, bytes.NewBuffer(payloadJSON))
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		return // Silently fail
	}
	defer resp.Body.Close()
}

// SetupDirectLokiLogging configures slog to send logs directly to Loki
func SetupDirectLokiLogging(lokiURL string, serviceName string) error {
	config := LokiConfig{
		URL: lokiURL,
		Labels: map[string]string{
			"service": serviceName,
			"job":     "go-direct",
			"source":  "application",
		},
		Level: slog.LevelInfo,
	}

	handler := NewLokiHandler(config)
	logger := slog.New(handler)
	slog.SetDefault(logger)

	slog.Info("Direct Loki logging configured",
		slog.String("loki_url", lokiURL),
		slog.String("service", serviceName))

	return nil
}
