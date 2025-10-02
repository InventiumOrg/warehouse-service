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
  "go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

type Server struct {
  router       *gin.Engine
  routes       *routes.Route
  db           *pgx.Conn
  otelShutdown func(context.Context) error
}

func NewServer(db *pgx.Conn) *Server {
  server := &Server{
    router: gin.Default(),
    db:     db,
}

  // Initialize OpenTelemetry
  ctx := context.Background()
  otelShutdown, err := observability.SetupOTelSDK(ctx)
  if err != nil {
    slog.Error("Failed to setup OpenTelemetry", slog.Any("err", err))
  } else {
    server.otelShutdown = otelShutdown
    slog.Info("OpenTelemetry initialized successfully")
  }

  server.routes = routes.NewRoute(db)
  return server
  }

  func (s *Server) Run(addr string) error {
  s.router.SetTrustedProxies(nil)

  // Add OpenTelemetry middleware
  s.router.Use(otelgin.Middleware("warehouse-service"))

  s.router.Use(cors.New(cors.Config{
    AllowOrigins:     []string{"http://localhost:3000"},
    AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
    AllowHeaders:     []string{"Origins", "Content-Type", "Authorization", "Bearer"},
    AllowCredentials: true,
    MaxAge:           12 * time.Hour,
  }))

  // Add health check routes (no auth required)
  s.routes.AddHealthRoutes(s.router)
  
  // Add business logic routes
  s.routes.AddWarehouseRoutes(s.router)
  s.routes.AddStorageRoomRoutes(s.router)
  
  return s.router.Run(addr)
}

// Shutdown gracefully shuts down the server and OpenTelemetry
func (s *Server) Shutdown(ctx context.Context) error {
  if s.otelShutdown != nil {
    if err := s.otelShutdown(ctx); err != nil {
      slog.Error("Failed to shutdown OpenTelemetry", slog.Any("err", err))
      return err
    }
    slog.Info("OpenTelemetry shutdown successfully")
  }
  return nil
}
