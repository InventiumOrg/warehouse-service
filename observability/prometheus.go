package observability

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PrometheusMetrics holds all Prometheus metrics for the inventory service
type PrometheusMetrics struct {
	// HTTP metrics
	HTTPRequestsTotal       *prometheus.CounterVec
	HTTPRequestDuration     *prometheus.HistogramVec
	HTTPRequestsInFlight    prometheus.Gauge
	HTTPResponseStatusTotal *prometheus.CounterVec // New: HTTP status code metrics

	// Database metrics
	DBConnectionsActive prometheus.Gauge
	DBOperationDuration *prometheus.HistogramVec
	DBOperationErrors   *prometheus.CounterVec

	// Business metrics
	WarehouseOperationsTotal *prometheus.CounterVec
	WarehouseActive          prometheus.Gauge
	AuthenticationAttempts   *prometheus.CounterVec

	// System metrics (automatically collected by Prometheus client)
	// - go_* metrics (goroutines, memory, GC, etc.)
	// - process_* metrics (CPU, memory, file descriptors, etc.)
}

// NewPrometheusMetrics creates and registers all Prometheus metrics
func NewPrometheusMetrics(serviceName string) *PrometheusMetrics {
	metrics := &PrometheusMetrics{
		// HTTP metrics following Prometheus naming conventions
		HTTPRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status_code"},
		),
		HTTPRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets, // Default buckets: .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10
			},
			[]string{"method", "endpoint"},
		),
		HTTPRequestsInFlight: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "http_requests_in_flight",
				Help: "Current number of HTTP requests being processed",
			},
		),
		// New: HTTP status code metrics grouped by status class (2xx, 4xx, 5xx)
		HTTPResponseStatusTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_response_status_total",
				Help: "Total number of HTTP responses by status class",
			},
			[]string{"method", "endpoint", "status_class"},
		),

		// Database metrics
		DBConnectionsActive: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "database_connections_active",
				Help: "Number of active database connections",
			},
		),
		DBOperationDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "database_operation_duration_seconds",
				Help:    "Database operation duration in seconds",
				Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5}, // Smaller buckets for DB ops
			},
			[]string{"operation", "table"},
		),
		DBOperationErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "database_operation_errors_total",
				Help: "Total number of database operation errors",
			},
			[]string{"operation", "table", "error_type"},
		),

		// Business metrics specific to inventory service
		WarehouseOperationsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "warehouse_operations_total",
				Help: "Total number of warehouse operations",
			},
			[]string{"operation", "category", "location"},
		),
		WarehouseActive: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "warehouse_active",
				Help: "Current number of active warehouse",
			},
		),
		AuthenticationAttempts: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "authentication_attempts_total",
				Help: "Total number of authentication attempts",
			},
			[]string{"status", "method"},
		),
	}

	// Register all metrics with Prometheus
	prometheus.MustRegister(
		metrics.HTTPRequestsTotal,
		metrics.HTTPRequestDuration,
		metrics.HTTPRequestsInFlight,
		metrics.HTTPResponseStatusTotal, // Register the new status code metric
		metrics.DBConnectionsActive,
		metrics.DBOperationDuration,
		metrics.DBOperationErrors,
		metrics.WarehouseOperationsTotal,
		metrics.WarehouseActive,
		metrics.AuthenticationAttempts,
	)

	slog.Info("Prometheus metrics registered", slog.String("service", serviceName))
	return metrics
}

// getStatusClass converts HTTP status code to status class (2xx, 4xx, 5xx, etc.)
func getStatusClass(statusCode int) string {
	switch {
	case statusCode >= 200 && statusCode < 300:
		return "2xx"
	case statusCode >= 300 && statusCode < 400:
		return "3xx"
	case statusCode >= 400 && statusCode < 500:
		return "4xx"
	case statusCode >= 500:
		return "5xx"
	default:
		return "1xx"
	}
}

// PrometheusMiddleware creates a Gin middleware for collecting HTTP metrics
func (m *PrometheusMetrics) PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip metrics collection for the metrics endpoint itself
		if c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}

		start := time.Now()

		// Increment in-flight requests
		m.HTTPRequestsInFlight.Inc()
		defer m.HTTPRequestsInFlight.Dec()

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start).Seconds()

		// Get route pattern (e.g., "/api/v1/warehouse/:id" instead of "/api/v1/warehouse/123")
		route := c.FullPath()
		if route == "" {
			route = "unknown"
		}

		// Record metrics
		statusCode := c.Writer.Status()
		statusClass := getStatusClass(statusCode)

		m.HTTPRequestsTotal.WithLabelValues(
			c.Request.Method,
			route,
			string(rune(statusCode)),
		).Inc()

		m.HTTPRequestDuration.WithLabelValues(
			c.Request.Method,
			route,
		).Observe(duration)

		// Record status class metrics (2xx, 4xx, 5xx)
		m.HTTPResponseStatusTotal.WithLabelValues(
			c.Request.Method,
			route,
			statusClass,
		).Inc()
	}
}

// RecordDBOperation records database operation metrics
func (m *PrometheusMetrics) RecordDBOperation(operation, table string, duration time.Duration, err error) {
	m.DBOperationDuration.WithLabelValues(operation, table).Observe(duration.Seconds())

	if err != nil {
		errorType := "unknown"
		// You can categorize errors here based on your needs
		// For example: "connection_error", "timeout", "constraint_violation", etc.
		m.DBOperationErrors.WithLabelValues(operation, table, errorType).Inc()
	}
}

// RecordInventoryOperation records business-specific inventory operations
func (m *PrometheusMetrics) RecordInventoryOperation(operation, category, location string) {
	m.WarehouseOperationsTotal.WithLabelValues(operation, category, location).Inc()
}

// UpdateInventoryCount updates the current count of active inventory items
func (m *PrometheusMetrics) UpdateInventoryCount(count float64) {
	m.WarehouseActive.Set(count)
}

// RecordAuthAttempt records authentication attempts
func (m *PrometheusMetrics) RecordAuthAttempt(status, method string) {
	m.AuthenticationAttempts.WithLabelValues(status, method).Inc()
}

// UpdateDBConnections updates the database connections gauge
func (m *PrometheusMetrics) UpdateDBConnections(count float64) {
	m.DBConnectionsActive.Set(count)
}

// RecordHTTPResponse records HTTP response metrics manually (if needed outside middleware)
func (m *PrometheusMetrics) RecordHTTPResponse(method, endpoint string, statusCode int, duration time.Duration) {
	statusClass := getStatusClass(statusCode)

	m.HTTPRequestsTotal.WithLabelValues(method, endpoint, string(rune(statusCode))).Inc()
	m.HTTPRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
	m.HTTPResponseStatusTotal.WithLabelValues(method, endpoint, statusClass).Inc()
}

// GetStatusCodeMetrics returns current status code counts (useful for debugging)
func (m *PrometheusMetrics) GetStatusCodeMetrics() map[string]float64 {
	// This is a helper function to get current metric values
	// Note: In production, you'd typically query these from Prometheus directly
	return map[string]float64{
		"2xx_responses": 0, // Placeholder - actual values would come from Prometheus
		"4xx_responses": 0,
		"5xx_responses": 0,
	}
}

// SetupPrometheusEndpoint adds the /metrics endpoint to the Gin router
func SetupPrometheusEndpoint(router *gin.Engine) {
	// Add the /metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	slog.Info("Prometheus metrics endpoint configured at /metrics")
}

// Example usage functions for common patterns

// WithDBMetrics wraps a database operation with automatic metrics collection
func (m *PrometheusMetrics) WithDBMetrics(operation, table string, fn func() error) error {
	start := time.Now()
	err := fn()
	m.RecordDBOperation(operation, table, time.Since(start), err)
	return err
}

// WithInventoryMetrics wraps an inventory operation with automatic metrics collection
func (m *PrometheusMetrics) WithInventoryMetrics(operation, category, location string, fn func() error) error {
	err := fn()
	if err == nil {
		m.RecordInventoryOperation(operation, category, location)
	}
	return err
}
