package observability

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// SetupOTelSDK bootstraps the OpenTelemetry pipeline for shipping to otel-collector.
// If it does not return an error, make sure to call shutdown for proper cleanup.
func SetupOTelSDK(ctx context.Context, serviceName, serviceVersion, otelCollectorEndpoint, otelHeaders string) (func(context.Context) error, error) {
	var shutdownFuncs []func(context.Context) error

	// shutdown calls cleanup functions registered via shutdownFuncs.
	shutdown := func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	// handleErr calls shutdown for cleanup and makes sure that all errors are returned.
	handleErr := func(inErr error) error {
		return errors.Join(inErr, shutdown(ctx))
	}

	// Create resource with service information
	res, err := newResource(serviceName, serviceVersion)
	if err != nil {
		return shutdown, handleErr(err)
	}

	// Set up propagator
	prop := newPropagator()
	otel.SetTextMapPropagator(prop)

	// Set up trace provider
	tracerProvider, err := newTracerProvider(ctx, res, otelCollectorEndpoint, otelHeaders)
	if err != nil {
		return shutdown, handleErr(err)
	}
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)

	// Set up meter provider
	meterProvider, err := newMeterProvider(ctx, res, otelCollectorEndpoint, otelHeaders)
	if err != nil {
		return shutdown, handleErr(err)
	}
	shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
	otel.SetMeterProvider(meterProvider)

	return shutdown, nil
}

func newResource(serviceName, serviceVersion string) (*resource.Resource, error) {
	// Create resource without merging to avoid schema conflicts
	return resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(serviceName),
		semconv.ServiceVersion(serviceVersion),
		semconv.ServiceInstanceID("warehouse-service"),
	), nil
}

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func newTracerProvider(ctx context.Context, res *resource.Resource, endpoint, headers string) (*trace.TracerProvider, error) {
	// Debug logging
	slog.Info("Configuring OTLP tracer",
		slog.String("endpoint", endpoint),
		slog.String("headers_raw", headers))

	// Parse headers from the environment variable format
	headerMap := make(map[string]string)
	if headers != "" {
		// Headers are in format "key1=value1,key2=value2"
		pairs := strings.Split(headers, ",")
		for _, pair := range pairs {
			if kv := strings.SplitN(pair, "=", 2); len(kv) == 2 {
				headerMap[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
			}
		}
		slog.Info("OTLP headers configured", slog.Int("header_count", len(headerMap)))
	}

	slog.Info("Using OTLP endpoint", slog.String("endpoint", endpoint))

	// For local collector, use simple endpoint configuration
	var err error

	// Configure the exporter with proper URL path for Grafana Cloud
	options := []otlptracehttp.Option{
		otlptracehttp.WithInsecure(),
		otlptracehttp.WithEndpoint(endpoint),
		// otlptracehttp.WithURLPath("/otlp/v1/traces"), // Grafana Cloud specific path
	}

	if len(headerMap) > 0 {
		options = append(options, otlptracehttp.WithHeaders(headerMap))
	}

	traceExporter, err := otlptracehttp.New(ctx, options...)
	if err != nil {
		return nil, err
	}

	tracerProvider := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter,
			trace.WithBatchTimeout(time.Second*5),
			trace.WithMaxExportBatchSize(512),
		),
		trace.WithResource(res),
		trace.WithSampler(trace.AlwaysSample()),
	)
	return tracerProvider, nil
}

func newMeterProvider(ctx context.Context, res *resource.Resource, endpoint, headers string) (*sdkmetric.MeterProvider, error) {
	// Debug logging
	slog.Info("Configuring OTLP metrics exporter", slog.String("endpoint", endpoint))

	// Parse headers from the environment variable format (if any)
	headerMap := make(map[string]string)
	if headers != "" {
		pairs := strings.Split(headers, ",")
		for _, pair := range pairs {
			if kv := strings.SplitN(pair, "=", 2); len(kv) == 2 {
				headerMap[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
			}
		}
	}

	// We need to add the OTLP metrics HTTP exporter
	// For now, let's create a basic meter provider that will work locally
	// TODO: Add OTLP metrics exporter when the import is available
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		// Add a periodic reader that exports every 30 seconds
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(
			// For now, we'll use a no-op exporter until we add the OTLP metrics exporter
			&noOpMetricExporter{},
			sdkmetric.WithInterval(30*time.Second),
		)),
	)

	slog.Info("Metrics provider configured (local only for now)")
	return meterProvider, nil
}

// Temporary no-op exporter until we add OTLP metrics support
type noOpMetricExporter struct{}

func (e *noOpMetricExporter) Temporality(sdkmetric.InstrumentKind) metricdata.Temporality {
	return metricdata.CumulativeTemporality
}

func (e *noOpMetricExporter) Aggregation(kind sdkmetric.InstrumentKind) sdkmetric.Aggregation {
	return sdkmetric.DefaultAggregationSelector(kind)
}

func (e *noOpMetricExporter) Export(context.Context, *metricdata.ResourceMetrics) error {
	// Log that metrics are being generated
	slog.Debug("Metrics exported (no-op)")
	return nil
}

func (e *noOpMetricExporter) ForceFlush(context.Context) error { return nil }
func (e *noOpMetricExporter) Shutdown(context.Context) error   { return nil }

// GetLogger returns a structured logger that integrates with OpenTelemetry
func GetLogger(name string) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})).With("service", name)
}

// SetupFileLogger configures slog to write JSON logs to a file
func SetupFileLogger(logFilePath string) error {
	// Create logs directory if it doesn't exist
	logDir := filepath.Dir(logFilePath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	// Open log file for writing (create if not exists, append if exists)
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	// Create a multi-writer to write to both stdout and file
	multiWriter := io.MultiWriter(os.Stdout, logFile)

	// Create JSON handler that writes to both stdout and file
	jsonHandler := slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true,
	})

	// Set the default logger
	logger := slog.New(jsonHandler)
	slog.SetDefault(logger)

	slog.Info("File logging configured",
		slog.String("log_file", logFilePath),
		slog.String("log_dir", logDir))

	return nil
}

// CreateMetrics creates and returns common application metrics
func CreateMetrics() (*AppMetrics, error) {
	meter := otel.Meter("warehouse-service")

	requestCounter, err := meter.Int64Counter(
		"http_requests_total",
		metric.WithDescription("Total number of HTTP requests"),
	)
	if err != nil {
		return nil, err
	}

	requestDuration, err := meter.Float64Histogram(
		"http_request_duration_seconds",
		metric.WithDescription("HTTP request duration in seconds"),
	)
	if err != nil {
		return nil, err
	}

	dbConnections, err := meter.Int64UpDownCounter(
		"db_connections_active",
		metric.WithDescription("Number of active database connections"),
	)
	if err != nil {
		return nil, err
	}

	return &AppMetrics{
		RequestCounter:  requestCounter,
		RequestDuration: requestDuration,
		DBConnections:   dbConnections,
	}, nil
}

// AppMetrics holds the application metrics
type AppMetrics struct {
	RequestCounter  metric.Int64Counter
	RequestDuration metric.Float64Histogram
	DBConnections   metric.Int64UpDownCounter
}

// CreateBusinessMetrics creates business-specific metrics for the warehouse service
func CreateBusinessMetrics() (*BusinessMetrics, error) {
	meter := otel.Meter("warehouse-service-business")

	// Warehouse Operations
	warehouseOperations, err := meter.Int64Counter(
		"warehouse_operations_total",
		metric.WithDescription("Total number of warehouse operations"),
	)
	if err != nil {
		return nil, err
	}

	warehouseCreated, err := meter.Int64Counter(
		"warehouse_created_total",
		metric.WithDescription("Total number of warehouses created"),
	)
	if err != nil {
		return nil, err
	}

	warehouseRetrievals, err := meter.Int64Counter(
		"warehouse_retrievals_total",
		metric.WithDescription("Total number of warehouse retrievals"),
	)
	if err != nil {
		return nil, err
	}

	warehouseRequests, err := meter.Int64Counter(
		"warehouse_list_requests_total",
		metric.WithDescription("Total number of warehouse list requests"),
	)
	if err != nil {
		return nil, err
	}

	warehouseDeletes, err := meter.Int64Counter(
		"warehouse_deletes_total",
		metric.WithDescription("Total number of warehouse deletions"),
	)
	if err != nil {
		return nil, err
	}

	// Storage Room Operations
	storageRoomOperations, err := meter.Int64Counter(
		"storage_room_operations_total",
		metric.WithDescription("Total number of storage room operations"),
	)
	if err != nil {
		return nil, err
	}

	storageRoomCreated, err := meter.Int64Counter(
		"storage_room_created_total",
		metric.WithDescription("Total number of storage rooms created"),
	)
	if err != nil {
		return nil, err
	}

	storageRoomRetrievals, err := meter.Int64Counter(
		"storage_room_retrievals_total",
		metric.WithDescription("Total number of storage room retrievals"),
	)
	if err != nil {
		return nil, err
	}

	storageRoomDeletes, err := meter.Int64Counter(
		"storage_room_deletes_total",
		metric.WithDescription("Total number of storage room deletions"),
	)
	if err != nil {
		return nil, err
	}

	// Database operation metrics
	dbOperationDuration, err := meter.Float64Histogram(
		"database_operation_duration_seconds",
		metric.WithDescription("Duration of database operations in seconds"),
	)
	if err != nil {
		return nil, err
	}

	dbOperationErrors, err := meter.Int64Counter(
		"database_operation_errors_total",
		metric.WithDescription("Total number of database operation errors"),
	)
	if err != nil {
		return nil, err
	}

	// Authentication metrics
	authenticationAttempts, err := meter.Int64Counter(
		"authentication_attempts_total",
		metric.WithDescription("Total number of authentication attempts"),
	)
	if err != nil {
		return nil, err
	}

	// Business logic metrics
	activeWarehouses, err := meter.Int64UpDownCounter(
		"active_warehouse_count",
		metric.WithDescription("Current number of active warehouses"),
	)
	if err != nil {
		return nil, err
	}

	activeStorageRooms, err := meter.Int64UpDownCounter(
		"active_storage_room_count",
		metric.WithDescription("Current number of active storage rooms"),
	)
	if err != nil {
		return nil, err
	}

	// Warehouse update metrics
	warehouseUpdates, err := meter.Int64Counter(
		"warehouse_updates_total",
		metric.WithDescription("Total number of warehouse updates"),
	)
	if err != nil {
		return nil, err
	}

	storageRoomUpdates, err := meter.Int64Counter(
		"storage_room_updates_total",
		metric.WithDescription("Total number of storage room updates"),
	)
	if err != nil {
		return nil, err
	}

	return &BusinessMetrics{
		WarehouseOperations:    warehouseOperations,
		WarehouseCreated:       warehouseCreated,
		WarehouseRetrievals:    warehouseRetrievals,
		WarehouseListRequests:  warehouseRequests,
		WarehouseDeletes:       warehouseDeletes,
		WarehouseUpdates:       warehouseUpdates,
		StorageRoomOperations:  storageRoomOperations,
		StorageRoomCreated:     storageRoomCreated,
		StorageRoomRetrievals:  storageRoomRetrievals,
		StorageRoomDeletes:     storageRoomDeletes,
		StorageRoomUpdates:     storageRoomUpdates,
		DBOperationDuration:    dbOperationDuration,
		DBOperationErrors:      dbOperationErrors,
		AuthenticationAttempts: authenticationAttempts,
		ActiveWarehouses:       activeWarehouses,
		ActiveStorageRooms:     activeStorageRooms,
	}, nil
}

// BusinessMetrics holds business-specific metrics
type BusinessMetrics struct {
	WarehouseOperations    metric.Int64Counter
	WarehouseCreated       metric.Int64Counter
	WarehouseRetrievals    metric.Int64Counter
	WarehouseListRequests  metric.Int64Counter
	WarehouseDeletes       metric.Int64Counter
	WarehouseUpdates       metric.Int64Counter
	StorageRoomOperations  metric.Int64Counter
	StorageRoomCreated     metric.Int64Counter
	StorageRoomRetrievals  metric.Int64Counter
	StorageRoomDeletes     metric.Int64Counter
	StorageRoomUpdates     metric.Int64Counter
	DBOperationDuration    metric.Float64Histogram
	DBOperationErrors      metric.Int64Counter
	AuthenticationAttempts metric.Int64Counter
	ActiveWarehouses       metric.Int64UpDownCounter
	ActiveStorageRooms     metric.Int64UpDownCounter
}

// LogConfig holds configuration for file logging
type LogConfig struct {
	FilePath   string
	MaxSizeMB  int64
	MaxBackups int
	MaxAgeDays int
	Compress   bool
}

// DefaultLogConfig returns a default logging configuration
func DefaultLogConfig() LogConfig {
	return LogConfig{
		FilePath:   "/logs/otel-logs.json",
		MaxSizeMB:  100, // 100MB per file
		MaxBackups: 5,   // Keep 5 backup files
		MaxAgeDays: 30,  // Keep logs for 30 days
		Compress:   true,
	}
}

// SetupAdvancedFileLogger configures slog with advanced file logging features
func SetupAdvancedFileLogger(config LogConfig) error {
	// Create logs directory if it doesn't exist
	logDir := filepath.Dir(config.FilePath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	// Open log file for writing (create if not exists, append if exists)
	logFile, err := os.OpenFile(config.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	// Create a multi-writer to write to both stdout and file
	multiWriter := io.MultiWriter(os.Stdout, logFile)

	// Create JSON handler with enhanced options
	jsonHandler := slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Add timestamp in ISO format
			if a.Key == slog.TimeKey {
				return slog.Attr{
					Key:   "timestamp",
					Value: slog.StringValue(time.Now().UTC().Format(time.RFC3339)),
				}
			}
			return a
		},
	})

	// Set the default logger
	logger := slog.New(jsonHandler)
	slog.SetDefault(logger)

	slog.Info("Advanced file logging configured",
		slog.String("log_file", config.FilePath),
		slog.String("log_dir", logDir),
		slog.Int64("max_size_mb", config.MaxSizeMB),
		slog.Int("max_backups", config.MaxBackups),
		slog.Int("max_age_days", config.MaxAgeDays))

	return nil
}
