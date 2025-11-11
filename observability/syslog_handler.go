package observability

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"log/syslog"
	"os"
)

// SyslogHandler implements slog.Handler to send logs via syslog
type SyslogHandler struct {
	writer   *syslog.Writer
	level    slog.Level
	fallback slog.Handler
}

// SyslogConfig holds configuration for syslog handler
type SyslogConfig struct {
	Network  string // "tcp", "udp", or "" for local
	Address  string // "localhost:514" or "" for local
	Priority syslog.Priority
	Tag      string
	Level    slog.Level
}

// NewSyslogHandler creates a new syslog handler
func NewSyslogHandler(config SyslogConfig) (*SyslogHandler, error) {
	var writer *syslog.Writer
	var err error

	if config.Network == "" && config.Address == "" {
		// Local syslog
		writer, err = syslog.New(config.Priority, config.Tag)
	} else {
		// Remote syslog
		writer, err = syslog.Dial(config.Network, config.Address, config.Priority, config.Tag)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create syslog writer: %w", err)
	}

	return &SyslogHandler{
		writer:   writer,
		level:    config.Level,
		fallback: slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: config.Level}),
	}, nil
}

// Enabled reports whether the handler handles records at the given level
func (h *SyslogHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

// Handle processes a log record
func (h *SyslogHandler) Handle(ctx context.Context, record slog.Record) error {
	// Also send to fallback (stdout)
	if err := h.fallback.Handle(ctx, record); err != nil {
		return err
	}

	// Create structured log entry
	logEntry := map[string]interface{}{
		"timestamp": record.Time.Format("2006-01-02T15:04:05.000Z07:00"),
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
		return err
	}

	// Send to syslog based on level
	switch record.Level {
	case slog.LevelDebug:
		return h.writer.Debug(string(logJSON))
	case slog.LevelInfo:
		return h.writer.Info(string(logJSON))
	case slog.LevelWarn:
		return h.writer.Warning(string(logJSON))
	case slog.LevelError:
		return h.writer.Err(string(logJSON))
	default:
		return h.writer.Info(string(logJSON))
	}
}

// WithAttrs returns a new handler with additional attributes
func (h *SyslogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &SyslogHandler{
		writer:   h.writer,
		level:    h.level,
		fallback: h.fallback.WithAttrs(attrs),
	}
}

// WithGroup returns a new handler with a group
func (h *SyslogHandler) WithGroup(name string) slog.Handler {
	return &SyslogHandler{
		writer:   h.writer,
		level:    h.level,
		fallback: h.fallback.WithGroup(name),
	}
}

// Close closes the syslog writer
func (h *SyslogHandler) Close() error {
	return h.writer.Close()
}

// SetupSyslogLogging configures slog to send logs via syslog
func SetupSyslogLogging(network, address, tag string) error {
	config := SyslogConfig{
		Network:  network,
		Address:  address,
		Priority: syslog.LOG_INFO | syslog.LOG_LOCAL0,
		Tag:      tag,
		Level:    slog.LevelInfo,
	}

	handler, err := NewSyslogHandler(config)
	if err != nil {
		return err
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	slog.Info("Syslog logging configured",
		slog.String("network", network),
		slog.String("address", address),
		slog.String("tag", tag))

	return nil
}
