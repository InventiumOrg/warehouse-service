package api

import (
	"context"
	"log/slog"
	"time"
	"warehouse-service/observability"
	routes "warehouse-service/routes"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type Server struct {
	router          *gin.Engine
	routes          *routes.Route
	db              *pgx.Conn
	otelShutdown    func(context.Context) error
	metrics         *observability.AppMetrics
	businessMetrics *observability.BusinessMetrics
}

func NewServer(db *pgx.Conn, serviceName, serviceVersion, otelEndpoint, otelHeaders string) *Server {
	// Setup OpenTelemetry
	ctx := context.Background()
	otelShutdown, err := observability.SetupOTelSDK(ctx, serviceName, serviceVersion, otelEndpoint, otelHeaders)
	if err != nil {
		slog.Error("Failed to setup OpenTelemetry", slog.Any("error", err))
		// Continue without OpenTelemetry
		otelShutdown = func(context.Context) error { return nil }
	}

	// Create metrics
	metrics, err := observability.CreateMetrics()
	if err != nil {
		slog.Error("Failed to create metrics", slog.Any("error", err))
	}

	// Create business metrics
	businessMetrics, err := observability.CreateBusinessMetrics()
	if err != nil {
		slog.Error("Failed to create business metrics", slog.Any("error", err))
	}

	router := gin.Default()

	// Add metrics middleware
	server := &Server{
		router:          router,
		db:              db,
		otelShutdown:    otelShutdown,
		metrics:         metrics,
		businessMetrics: businessMetrics,
	}

	// Add middleware
	router.Use(server.metricsMiddleware())
	server.router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origins", "Content-Type", "Authorization", "Bearer"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	// Setup routes
	server.routes = routes.NewRoute(db, businessMetrics)

	return server
}

func (s *Server) Run(addr string, serviceName string) error {
	slog.Info("Starting warehouse service server",
		slog.String("address", addr),
		slog.String("service", serviceName))

	s.router.SetTrustedProxies(nil)

	// Add health check routes (no auth required)
	s.routes.AddHealthRoutes(s.router)

	// Add business logic routes
	s.routes.AddWarehouseRoutes(s.router)
	s.routes.AddStorageRoomRoutes(s.router)

	return s.router.Run(addr)
}

// Shutdown gracefully shuts down the server and OpenTelemetry
func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("Shutting down warehouse service server")

	if s.otelShutdown != nil {
		if err := s.otelShutdown(ctx); err != nil {
			slog.Error("Failed to shutdown OpenTelemetry", slog.Any("error", err))
			return err
		}
	}

	if s.db != nil {
		s.db.Close(ctx)
	}

	return nil
}

// metricsMiddleware records HTTP request metrics
func (s *Server) metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Record metrics if available
		if s.metrics != nil {
			duration := time.Since(start).Seconds()

			// Record request counter
			s.metrics.RequestCounter.Add(c.Request.Context(), 1,
				metric.WithAttributes(
					attribute.String("method", c.Request.Method),
					attribute.String("route", c.FullPath()),
					attribute.Int("status_code", c.Writer.Status()),
				))

			// Record request duration
			s.metrics.RequestDuration.Record(c.Request.Context(), duration,
				metric.WithAttributes(
					attribute.String("method", c.Request.Method),
					attribute.String("route", c.FullPath()),
					attribute.Int("status_code", c.Writer.Status()),
				))
		}
	}
}
